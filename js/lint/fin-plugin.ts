import path from "node:path";

import { definePlugin, defineRule } from "@oxlint/plugins";
import type { Context, ESTree } from "@oxlint/plugins";

const projectRoot = path.resolve(import.meta.dirname, "..");

const effectHooks: Record<string, true> = {
  useEffect: true,
  useLayoutEffect: true,
  useInsertionEffect: true,
};

const privateDirs: Record<string, true> = {
  _components: true,
  _hooks: true,
};

type SourceNode = ESTree.Expression | null;

function projectPath(context: Context): string {
  return path.relative(projectRoot, context.filename).split(path.sep).join("/");
}

function hookName(call: ESTree.CallExpression): string | null {
  const { callee } = call;

  if (callee.type === "Identifier") {
    return callee.name;
  }

  if (
    callee.type === "MemberExpression" &&
    callee.object.type === "Identifier" &&
    callee.object.name === "React" &&
    callee.property.type === "Identifier"
  ) {
    return callee.property.name;
  }

  return null;
}

function onModuleSources(check: (source: string, node: ESTree.Node) => void) {
  const visit = (node: ESTree.Node, source: SourceNode) => {
    if (source?.type === "Literal" && typeof source.value === "string") {
      check(source.value, node);
    }
  };

  return {
    ImportDeclaration: (node: ESTree.ImportDeclaration) => visit(node, node.source),
    ExportNamedDeclaration: (node: ESTree.ExportNamedDeclaration) => visit(node, node.source),
    ExportAllDeclaration: (node: ESTree.ExportAllDeclaration) => visit(node, node.source),
    ImportExpression: (node: ESTree.ImportExpression) => visit(node, node.source),
  };
}

const aliasImports = defineRule({
  meta: {
    type: "problem",
    docs: { description: "Require @/ alias imports instead of relative paths." },
  },
  create(context) {
    return onModuleSources((source, node) => {
      if (source.startsWith(".")) {
        context.report({ node, message: `Use an "@/" import instead of "${source}".` });
      }
    });
  },
});

const noComments = defineRule({
  meta: {
    type: "suggestion",
    docs: { description: "Disallow comments; use names and small functions instead." },
  },
  create(context) {
    return {
      Program() {
        for (const comment of context.sourceCode.getAllComments()) {
          context.report({ node: comment, message: "Comments are not allowed." });
        }
      },
    };
  },
});

const routePrivateImports = defineRule({
  meta: {
    type: "problem",
    docs: { description: "Keep _components/ and _hooks/ private to their route subtree." },
  },
  create(context) {
    const file = projectPath(context);
    const fileWithoutExtension = file.slice(0, file.length - path.extname(file).length);

    const checkSources = onModuleSources((source, node) => {
      if (!source.startsWith("@/")) {
        return;
      }

      const segments = source.split("/");
      const privateIndex = segments.findIndex((segment) => privateDirs[segment] === true);

      if (privateIndex === -1) {
        return;
      }

      const owner = ["src", ...segments.slice(1, privateIndex)].join("/");
      const insideOwner = file.startsWith(`${owner}/`) || fileWithoutExtension === owner;
      const rootLayout =
        file === "src/router.ts" &&
        (source === "@/app/_components/authenticated-layout" ||
          source === "@/app/_components/root-layout");

      if (!insideOwner && !rootLayout) {
        context.report({ node, message: `"${source}" is private to ${owner}/.` });
      }
    });

    return {
      ...checkSources,
      Program(node: ESTree.Program) {
        const inPrivateDir = file.split("/").some((segment) => privateDirs[segment] === true);

        if (inPrivateDir && !file.startsWith("src/app/")) {
          context.report({
            node,
            message: "_components/ and _hooks/ directories belong under src/app/ only.",
          });
        }
      },
    };
  },
});

const noEffectsOutsideHooks = defineRule({
  meta: {
    type: "problem",
    docs: { description: "Allow effect hooks only inside src/lib/hooks/." },
  },
  create(context) {
    if (projectPath(context).startsWith("src/lib/hooks/")) {
      return {};
    }

    return {
      CallExpression(node) {
        const name = hookName(node);

        if (name !== null && effectHooks[name] === true) {
          context.report({
            node,
            message: `${name} is only allowed in src/lib/hooks/; write a hook there.`,
          });
        }
      },
    };
  },
});

const maxOneUseState = defineRule({
  meta: {
    type: "suggestion",
    docs: { description: "Allow at most one useState per component; extract a hook instead." },
  },
  create(context) {
    const file = projectPath(context);

    const exempt =
      file.startsWith("src/lib/hooks/") ||
      file.startsWith("src/lib/queries/") ||
      file.split("/").includes("_hooks");

    if (exempt) {
      return {};
    }

    const counts: number[] = [];

    const enter = () => {
      counts.push(0);
    };

    const exit = () => {
      counts.pop();
    };

    return {
      FunctionDeclaration: enter,
      "FunctionDeclaration:exit": exit,
      FunctionExpression: enter,
      "FunctionExpression:exit": exit,
      ArrowFunctionExpression: enter,
      "ArrowFunctionExpression:exit": exit,
      CallExpression(node) {
        const top = counts.length - 1;

        if (hookName(node) !== "useState" || top < 0) {
          return;
        }

        const count = (counts[top] ?? 0) + 1;

        counts[top] = count;

        if (count > 1) {
          context.report({
            node,
            message: "Only one useState per component; move related state into a composite hook.",
          });
        }
      },
    };
  },
});

export default definePlugin({
  meta: { name: "fin" },
  rules: {
    "alias-imports": aliasImports,
    "no-comments": noComments,
    "route-private-imports": routePrivateImports,
    "no-effects-outside-hooks": noEffectsOutsideHooks,
    "max-one-use-state": maxOneUseState,
  },
});
