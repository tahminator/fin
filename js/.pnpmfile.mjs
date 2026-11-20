function readPackage(pkg) {
  if (pkg.name === "openapi-typescript") {
    delete pkg.peerDependencies.typescript;

    pkg.dependencies.typescript = "^5.9.3";
  }

  return pkg;
}

export const hooks = { readPackage };
