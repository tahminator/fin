package service

import (
	"regexp"
	"slices"
	"strings"

	"github.com/tahminator/fin/internal/domain"
)

var (
	trademarkSymbols      = strings.NewReplacer("®", "", "™", "")
	nonAlphanumeric       = regexp.MustCompile(`[^a-z0-9]+`)
	supplementaryCardWord = regexp.MustCompile(`\b(?:additional|authorized user|supplementary)\b`)
	repeatedSpaces        = regexp.MustCompile(`\s+`)
	genericCardWords      = []string{"account", "card", "checking", "credit", "debit", "mastercard", "the", "visa"}
)

type cardName struct {
	text        string
	words       []string
	distinctive []string
}

type cardMatcher struct {
	aliases []cardName
	issuer  string
}

func normalizeCardName(value string) string {
	value = trademarkSymbols.Replace(strings.ToLower(value))

	return strings.TrimSpace(nonAlphanumeric.ReplaceAllString(value, " "))
}

func newCardName(text string) cardName {
	words := strings.Fields(text)

	distinctive := slices.DeleteFunc(slices.Clone(words), func(word string) bool {
		return slices.Contains(genericCardWords, word)
	})

	return cardName{text: text, words: words, distinctive: distinctive}
}

func cardSearchName(account domain.Account) string {
	if account.OfficialName != nil {
		if name := strings.TrimSpace(*account.OfficialName); name != "" {
			return name
		}
	}

	return strings.TrimSpace(account.Name)
}

func cardNameAliases(name string) []string {
	normalized := normalizeCardName(name)
	stripped := strings.TrimSpace(repeatedSpaces.ReplaceAllString(supplementaryCardWord.ReplaceAllString(normalized, ""), " "))

	if stripped == "" || stripped == normalized {
		return []string{normalized}
	}

	return []string{normalized, stripped}
}

func cardSearchTerms(name string) []string {
	return append([]string{name}, cardNameAliases(name)[1:]...)
}

func newCardMatcher(account domain.Account) cardMatcher {
	aliases := cardNameAliases(cardSearchName(account))
	names := make([]cardName, len(aliases))

	for i, alias := range aliases {
		names[i] = newCardName(alias)
	}

	issuer := ""
	if account.InstitutionName != nil {
		issuer = normalizeCardName(*account.InstitutionName)
	}

	return cardMatcher{aliases: names, issuer: issuer}
}

func (m cardMatcher) best(cards []domain.CatalogCard) *domain.CatalogCard {
	var best *domain.CatalogCard

	bestScore := 0

	for i := range cards {
		if score := m.score(cards[i]); score > bestScore {
			best = &cards[i]
			bestScore = score
		}
	}

	return best
}

func (m cardMatcher) score(card domain.CatalogCard) int {
	candidate := newCardName(normalizeCardName(card.Name))
	nameScore := 0

	for _, alias := range m.aliases {
		nameScore = max(nameScore, scoreCardName(alias, candidate))
	}

	if nameScore == 0 {
		return 0
	}

	return nameScore + m.scoreIssuer(card.Issuer)
}

func (m cardMatcher) scoreIssuer(issuer *string) int {
	if m.issuer == "" || issuer == nil {
		return 0
	}

	candidate := normalizeCardName(*issuer)
	if candidate == "" {
		return 0
	}

	if strings.Contains(candidate, m.issuer) || strings.Contains(m.issuer, candidate) {
		return 50
	}

	return 0
}

func scoreCardName(alias, candidate cardName) int {
	matching := 0

	for _, word := range slices.Compact(slices.Sorted(slices.Values(candidate.distinctive))) {
		if slices.Contains(alias.distinctive, word) {
			matching++
		}
	}

	switch {
	case matching == 0:
		return 0
	case candidate.text == alias.text:
		return 1000
	case containsWords(candidate.words, alias.words) || containsWords(alias.words, candidate.words):
		return 700
	}

	return matching*100 - min(abs(len(candidate.text)-len(alias.text)), 99)
}

func containsWords(container, words []string) bool {
	for _, word := range words {
		if !slices.Contains(container, word) {
			return false
		}
	}

	return true
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}
