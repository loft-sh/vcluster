package pluralize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Rule -- pluralize rule expression and replacement value.
type Rule struct {
	expression  *regexp.Regexp
	replacement string
}

// Client -- pluralize client.
type Client struct {
	pluralRules      []Rule
	singularRules    []Rule
	uncountables     map[string]bool
	irregularSingles map[string]string
	irregularPlurals map[string]string
	interpolateExpr  *regexp.Regexp
}

// NewClient - pluralization client factory method.
func NewClient() *Client {
	client := Client{}
	client.init()

	return &client
}

func (c *Client) init() {
	c.pluralRules = make([]Rule, 0)
	c.singularRules = make([]Rule, 0)
	c.uncountables = make(map[string]bool)
	c.irregularSingles = make(map[string]string)
	c.irregularPlurals = make(map[string]string)

	c.loadIrregularRules()
	c.loadPluralizationRules()
	c.loadSingularizationRules()
	c.loadUncountableRules()
	c.interpolateExpr = regexp.MustCompile(`\$(\d{1,2})`)
}

// Pluralize -- Pluralize or singularize a word based on the passed in count.
// 	word: the word to pluralize
// 	count: how many of the word exist
// 	inclusive: whether to prefix with the number (e.g. 3 ducks)
func (c *Client) Pluralize(word string, count int, inclusive bool) string {
	pluralized := func() func(string) string {
		if count == 1 {
			return c.Singular
		}

		return c.Plural
	}

	if inclusive {
		return fmt.Sprintf("%d %s", count, pluralized()(word))
	}

	return pluralized()(word)
}

// Plural -- Pluralize a word.
func (c *Client) Plural(word string) string {
	return c.replaceWord(c.irregularSingles, c.irregularPlurals, c.pluralRules)(word)
}

// IsPlural -- Check if a word is plural.
func (c *Client) IsPlural(word string) bool {
	return c.checkWord(c.irregularSingles, c.irregularPlurals, c.pluralRules)(word)
}

// Singular -- Singularize a word.
func (c *Client) Singular(word string) string {
	return c.replaceWord(c.irregularPlurals, c.irregularSingles, c.singularRules)(word)
}

// IsSingular -- Check if a word is singular.
func (c *Client) IsSingular(word string) bool {
	return c.checkWord(c.irregularPlurals, c.irregularSingles, c.singularRules)(word)
}

// AddPluralRule -- Add a pluralization rule to the collection.
func (c *Client) AddPluralRule(rule string, replacement string) {
	c.pluralRules = append(c.pluralRules, Rule{sanitizeRule(rule), replacement})
}

// AddSingularRule -- Add a singularization rule to the collection.
func (c *Client) AddSingularRule(rule string, replacement string) {
	c.singularRules = append(c.singularRules, Rule{sanitizeRule(rule), replacement})
}

// AddUncountableRule -- Add an uncountable word rule.
func (c *Client) AddUncountableRule(word string) {
	if !isExpr(word) {
		c.uncountables[strings.ToLower(word)] = true
		return
	}

	c.AddPluralRule(word, `$0`)
	c.AddSingularRule(word, `$0`)
}

// AddIrregularRule -- Add an irregular word definition.
func (c *Client) AddIrregularRule(single string, plural string) {
	p := strings.ToLower(plural)
	s := strings.ToLower(single)

	c.irregularSingles[s] = p
	c.irregularPlurals[p] = s
}

func (c *Client) replaceWord(replaceMap map[string]string, keepMap map[string]string, rules []Rule) func(w string) string { //nolint:lll
	f := func(word string) string {
		// Get the correct token and case restoration functions.
		var token = strings.ToLower(word)

		// Check against the keep object map.
		if _, ok := keepMap[token]; ok {
			return restoreCase(word, token)
		}

		// Check against the replacement map for a direct word replacement.
		if replaceToken, ok := replaceMap[token]; ok {
			return restoreCase(word, replaceToken)
		}

		// Run all the rules against the word.
		return c.sanitizeWord(token, word, rules)
	}

	return f
}

func (c *Client) checkWord(replaceMap map[string]string, keepMap map[string]string, rules []Rule) func(w string) bool {
	f := func(word string) bool {
		var token = strings.ToLower(word)

		if _, ok := keepMap[token]; ok {
			return true
		}

		if _, ok := replaceMap[token]; ok {
			return false
		}

		return c.sanitizeWord(token, token, rules) == token
	}

	return f
}

func (c *Client) interpolate(str string, args []string) string {
	lookup := map[string]string{}

	for _, submatch := range c.interpolateExpr.FindAllStringSubmatch(str, -1) {
		element, _ := strconv.Atoi(submatch[1])
		lookup[submatch[0]] = args[element]
	}

	result := c.interpolateExpr.ReplaceAllStringFunc(str, func(repl string) string {
		return lookup[repl]
	})

	return result
}

func (c *Client) replace(word string, rule Rule) string {
	return rule.expression.ReplaceAllStringFunc(word, func(w string) string {
		match := rule.expression.FindString(word)
		index := rule.expression.FindStringIndex(word)[0]
		args := rule.expression.FindAllStringSubmatch(word, -1)[0]

		result := c.interpolate(rule.replacement, args)

		if match == `` {
			return restoreCase(word[index-1:index], result)
		}
		return restoreCase(match, result)
	})
}

func (c *Client) sanitizeWord(token string, word string, rules []Rule) string {
	// If empty string
	if len(token) == 0 {
		return word
	}
	// If does not need fixup
	if _, ok := c.uncountables[token]; ok {
		return word
	}

	// Iterate over the sanitization rules and use the first one to match.
	// NOTE: iterate rules array in reverse order specific => general rules
	for i := len(rules) - 1; i >= 0; i-- {
		if rules[i].expression.MatchString(word) {
			return c.replace(word, rules[i])
		}
	}

	return word
}

func sanitizeRule(rule string) *regexp.Regexp {
	if isExpr(rule) {
		return regexp.MustCompile(rule)
	}

	return regexp.MustCompile(`(?i)^` + rule + `$`)
}

func restoreCase(word string, token string) string {
	// Tokens are an exact match.
	if word == token {
		return token
	}

	// Lower cased words. E.g. "hello".
	if word == strings.ToLower(word) {
		return strings.ToLower(token)
	}

	// Upper cased words. E.g. "WHISKY".
	if word == strings.ToUpper(word) {
		return strings.ToUpper(token)
	}

	// Title cased words. E.g. "Title".
	if word[:1] == strings.ToUpper(word[:1]) {
		return strings.ToUpper(token[:1]) + strings.ToLower(token[1:])
	}

	// Lower cased words. E.g. "test".
	return strings.ToLower(token)
}

// isExpr -- helper to detect if string represents an expression by checking first character to be `(`.
func isExpr(s string) bool {
	return s[:1] == `(`
}

func (c *Client) loadIrregularRules() { //nolint:funlen
	var irregularRules = []struct {
		single string
		plural string
	}{
		// Pronouns.
		{`I`, `we`},
		{`me`, `us`},
		{`he`, `they`},
		{`she`, `they`},
		{`them`, `them`},
		{`myself`, `ourselves`},
		{`yourself`, `yourselves`},
		{`itself`, `themselves`},
		{`herself`, `themselves`},
		{`himself`, `themselves`},
		{`themself`, `themselves`},
		{`is`, `are`},
		{`was`, `were`},
		{`has`, `have`},
		{`this`, `these`},
		{`that`, `those`},
		{`my`, `our`},
		{`its`, `their`},
		{`his`, `their`},
		{`her`, `their`},
		// Words ending in with a consonant and `o`.
		{`echo`, `echoes`},
		{`dingo`, `dingoes`},
		{`volcano`, `volcanoes`},
		{`tornado`, `tornadoes`},
		{`torpedo`, `torpedoes`},
		// Ends with `us`.
		{`genus`, `genera`},
		{`viscus`, `viscera`},
		// Ends with `ma`.
		{`stigma`, `stigmata`},
		{`stoma`, `stomata`},
		{`dogma`, `dogmata`},
		{`lemma`, `lemmata`},
		{`schema`, `schemata`},
		{`anathema`, `anathemata`},
		// Other irregular rules.
		{`ox`, `oxen`},
		{`axe`, `axes`},
		{`die`, `dice`},
		{`yes`, `yeses`},
		{`foot`, `feet`},
		{`eave`, `eaves`},
		{`goose`, `geese`},
		{`tooth`, `teeth`},
		{`quiz`, `quizzes`},
		{`human`, `humans`},
		{`proof`, `proofs`},
		{`carve`, `carves`},
		{`valve`, `valves`},
		{`looey`, `looies`},
		{`thief`, `thieves`},
		{`groove`, `grooves`},
		{`pickaxe`, `pickaxes`},
		{`passerby`, `passersby`},
		{`canvas`, `canvases`},
		{`sms`, `sms`},
	}

	for _, r := range irregularRules {
		c.AddIrregularRule(r.single, r.plural)
	}
}

func (c *Client) loadPluralizationRules() {
	var pluralizationRules = []struct {
		rule        string
		replacement string
	}{
		{`(?i)s?$`, `s`},
		{`(?i)[^[:ascii:]]$`, `$0`},
		{`(?i)([^aeiou]ese)$`, `$1`},
		{`(?i)(ax|test)is$`, `$1es`},
		{`(?i)(alias|[^aou]us|t[lm]as|gas|ris)$`, `$1es`},
		{`(?i)(e[mn]u)s?$`, `$1s`},
		{`(?i)([^l]ias|[aeiou]las|[ejzr]as|[iu]am)$`, `$1`},
		{`(?i)(alumn|syllab|vir|radi|nucle|fung|cact|stimul|termin|bacill|foc|uter|loc|strat)(?:us|i)$`, `$1i`}, //nolint:lll,misspell
		{`(?i)(alumn|alg|vertebr)(?:a|ae)$`, `$1ae`},
		{`(?i)(seraph|cherub)(?:im)?$`, `$1im`},
		{`(?i)(her|at|gr)o$`, `$1oes`},
		{`(?i)(agend|addend|millenni|dat|extrem|bacteri|desiderat|strat|candelabr|errat|ov|symposi|curricul|automat|quor)(?:a|um)$`, `$1a`}, //nolint:lll,misspell
		{`(?i)(apheli|hyperbat|periheli|asyndet|noumen|phenomen|criteri|organ|prolegomen|hedr|automat)(?:a|on)$`, `$1a`},
		{`(?i)sis$`, `ses`},
		{`(?i)(?:(kni|wi|li)fe|(ar|l|ea|eo|oa|hoo)f)$`, `$1$2ves`},
		{`(?i)([^aeiouy]|qu)y$`, `$1ies`},
		{`(?i)([^ch][ieo][ln])ey$`, `$1ies`},
		{`(?i)(x|ch|ss|sh|zz)$`, `$1es`},
		{`(?i)(matr|cod|mur|sil|vert|ind|append)(?:ix|ex)$`, `$1ices`},
		{`(?i)\b((?:tit)?m|l)(?:ice|ouse)$`, `$1ice`},
		{`(?i)(pe)(?:rson|ople)$`, `$1ople`},
		{`(?i)(child)(?:ren)?$`, `$1ren`},
		{`(?i)eaux$`, `$0`},
		{`(?i)m[ae]n$`, `men`},
		{`thou`, `you`},
	}

	for _, r := range pluralizationRules {
		c.AddPluralRule(r.rule, r.replacement)
	}
}

func (c *Client) loadSingularizationRules() {
	var singularizationRules = []struct {
		rule        string
		replacement string
	}{
		{`(?i)s$`, ``},
		{`(?i)(ss)$`, `$1`},
		{`(?i)(wi|kni|(?:after|half|high|low|mid|non|night|[^\w]|^)li)ves$`, `$1fe`},
		{`(?i)(ar|(?:wo|[ae])l|[eo][ao])ves$`, `$1f`},
		{`(?i)ies$`, `y`},
		{`(?i)(dg|ss|ois|lk|ok|wn|mb|th|ch|ec|oal|is|ck|ix|sser|ts|wb)ies$`, `$1ie`},
		{`(?i)\b(l|(?:neck|cross|hog|aun)?t|coll|faer|food|gen|goon|group|hipp|junk|vegg|(?:pork)?p|charl|calor|cut)ies$`, `$1ie`}, //nolint:lll
		{`(?i)\b(mon|smil)ies$`, `$1ey`},
		{`(?i)\b((?:tit)?m|l)ice$`, `$1ouse`},
		{`(?i)(seraph|cherub)im$`, `$1`},
		{`(?i)(x|ch|ss|sh|zz|tto|go|cho|alias|[^aou]us|t[lm]as|gas|(?:her|at|gr)o|[aeiou]ris)(?:es)?$`, `$1`},
		{`(?i)(analy|diagno|parenthe|progno|synop|the|empha|cri|ne)(?:sis|ses)$`, `$1sis`},
		{`(?i)(movie|twelve|abuse|e[mn]u)s$`, `$1`},
		{`(?i)(test)(?:is|es)$`, `$1is`},
		{`(?i)(alumn|syllab|vir|radi|nucle|fung|cact|stimul|termin|bacill|foc|uter|loc|strat)(?:us|i)$`, `$1us`},              //nolint:lll,misspell
		{`(?i)(agend|addend|millenni|dat|extrem|bacteri|desiderat|strat|candelabr|errat|ov|symposi|curricul|quor)a$`, `$1um`}, //nolint:lll,misspell
		{`(?i)(apheli|hyperbat|periheli|asyndet|noumen|phenomen|criteri|organ|prolegomen|hedr|automat)a$`, `$1on`},
		{`(?i)(alumn|alg|vertebr)ae$`, `$1a`},
		{`(?i)(cod|mur|sil|vert|ind)ices$`, `$1ex`},
		{`(?i)(matr|append)ices$`, `$1ix`},
		{`(?i)(pe)(rson|ople)$`, `$1rson`},
		{`(?i)(child)ren$`, `$1`},
		{`(?i)(eau)x?$`, `$1`},
		{`(?i)men$`, `man`},
	}

	for _, r := range singularizationRules {
		c.AddSingularRule(r.rule, r.replacement)
	}
}

func (c *Client) loadUncountableRules() { //nolint:funlen
	var uncountableRules = []string{
		// Singular words with no plurals.
		`adulthood`,
		`advice`,
		`agenda`,
		`aid`,
		`aircraft`,
		`alcohol`,
		`ammo`,
		`analytics`,
		`anime`,
		`athletics`,
		`audio`,
		`bison`,
		`blood`,
		`bream`,
		`buffalo`,
		`butter`,
		`carp`,
		`cash`,
		`chassis`,
		`chess`,
		`clothing`,
		`cod`,
		`commerce`,
		`cooperation`,
		`corps`,
		`debris`,
		`diabetes`,
		`digestion`,
		`elk`,
		`energy`,
		`equipment`,
		`excretion`,
		`expertise`,
		`firmware`,
		`flounder`,
		`fun`,
		`gallows`,
		`garbage`,
		`graffiti`,
		`hardware`,
		`headquarters`,
		`health`,
		`herpes`,
		`highjinks`,
		`homework`,
		`housework`,
		`information`,
		`jeans`,
		`justice`,
		`kudos`,
		`labour`,
		`literature`,
		`machinery`,
		`mackerel`,
		`mail`,
		`media`,
		`mews`,
		`moose`,
		`music`,
		`mud`,
		`manga`,
		`news`,
		`only`,
		`personnel`,
		`pike`,
		`plankton`,
		`pliers`,
		`police`,
		`pollution`,
		`premises`,
		`rain`,
		`research`,
		`rice`,
		`salmon`,
		`scissors`,
		`series`,
		`sewage`,
		`shambles`,
		`shrimp`,
		`software`,
		`staff`,
		`swine`,
		`tennis`,
		`traffic`,
		`transportation`,
		`trout`,
		`tuna`,
		`wealth`,
		`welfare`,
		`whiting`,
		`wildebeest`,
		`wildlife`,
		`you`,
		// Regexes.
		`(?i)pok[eé]mon$`,  //
		`(?i)[^aeiou]ese$`, // "chinese", "japanese"
		`(?i)deer$`,        // "deer", "reindeer"
		`(?i)(fish)$`,      // "fish", "blowfish", "angelfish"
		`(?i)measles$`,     //
		`(?i)o[iu]s$`,      // "carnivorous"
		`(?i)pox$`,         // "chickpox", "smallpox"
		`(?i)sheep$`,       //
	}

	for _, w := range uncountableRules {
		c.AddUncountableRule(w)
	}
}
