package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleGrammar
	ruleSelector
	ruleSimpleSelector
	ruleSimpleSelectorOp
	ruleCompoundSelector
	ruleFunctionSelector
	ruleFunction
	ruleFunctionOp
	ruleSource
	ruleNamespace
	ruleNamespacePart
	ruleWildcard
	ruleCriteria
	ruleMultiCriteria
	ruleCompoundCriteria
	ruleSimpleCriteria
	ruleIdCriteria
	rulePublisherCriteria
	ruleSourceCriteria
	ruleTimeCriteria
	ruleBoolean
	ruleBooleanOp
	ruleStatementId
	rulePublisherId
	ruleComparison
	ruleComparisonOp
	ruleLimit
	ruleUInt
	ruleWS
	ruleWSX
	ruleWhiteSpace
	ruleEOL
	ruleEOF
	rulePegText

	rulePre
	ruleIn
	ruleSuf
)

var rul3s = [...]string{
	"Unknown",
	"Grammar",
	"Selector",
	"SimpleSelector",
	"SimpleSelectorOp",
	"CompoundSelector",
	"FunctionSelector",
	"Function",
	"FunctionOp",
	"Source",
	"Namespace",
	"NamespacePart",
	"Wildcard",
	"Criteria",
	"MultiCriteria",
	"CompoundCriteria",
	"SimpleCriteria",
	"IdCriteria",
	"PublisherCriteria",
	"SourceCriteria",
	"TimeCriteria",
	"Boolean",
	"BooleanOp",
	"StatementId",
	"PublisherId",
	"Comparison",
	"ComparisonOp",
	"Limit",
	"UInt",
	"WS",
	"WSX",
	"WhiteSpace",
	"EOL",
	"EOF",
	"PegText",

	"Pre_",
	"_In_",
	"_Suf",
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(string(([]rune(buffer)[node.begin:node.end]))))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (node *node32) Print(buffer string) {
	node.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next uint32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: uint32(t.begin), end: uint32(t.end), next: uint32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = uint32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, uint32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: ruleIn, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: ruleSuf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth uint32, index int) {
	t.tree[index] = token32{pegRule: rule, begin: uint32(begin), end: uint32(end), next: uint32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

func (t *tokens32) Expand(index int) {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
}

type QueryParser struct {
	*Query

	Buffer string
	buffer []rune
	rules  [35]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	Pretty bool
	tokens32
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *QueryParser
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *QueryParser) PrintSyntaxTree() {
	p.tokens32.PrintSyntaxTree(p.Buffer)
}

func (p *QueryParser) Highlighter() {
	p.PrintSyntax()
}

func (p *QueryParser) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
		p.buffer = append(p.buffer, endSymbol)
	}

	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	var max token32
	position, depth, tokenIndex, buffer, _rules := uint32(0), uint32(0), 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin uint32) {
		tree.Expand(tokenIndex)
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position, depth}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 Grammar <- <('S' 'E' 'L' 'E' 'C' 'T' WS Selector WS Source (WS Criteria)? (WS Limit)? WSX EOF)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				if buffer[position] != rune('S') {
					goto l0
				}
				position++
				if buffer[position] != rune('E') {
					goto l0
				}
				position++
				if buffer[position] != rune('L') {
					goto l0
				}
				position++
				if buffer[position] != rune('E') {
					goto l0
				}
				position++
				if buffer[position] != rune('C') {
					goto l0
				}
				position++
				if buffer[position] != rune('T') {
					goto l0
				}
				position++
				if !_rules[ruleWS]() {
					goto l0
				}
				{
					position2 := position
					depth++
					{
						switch buffer[position] {
						case 'C':
							{
								position4 := position
								depth++
								{
									position5 := position
									depth++
									{
										position6 := position
										depth++
										{
											position7 := position
											depth++
											if buffer[position] != rune('C') {
												goto l0
											}
											position++
											if buffer[position] != rune('O') {
												goto l0
											}
											position++
											if buffer[position] != rune('U') {
												goto l0
											}
											position++
											if buffer[position] != rune('N') {
												goto l0
											}
											position++
											if buffer[position] != rune('T') {
												goto l0
											}
											position++
											depth--
											add(ruleFunctionOp, position7)
										}
										depth--
										add(rulePegText, position6)
									}
									depth--
									add(ruleFunction, position5)
								}
								if buffer[position] != rune('(') {
									goto l0
								}
								position++
								if !_rules[ruleSimpleSelector]() {
									goto l0
								}
								if buffer[position] != rune(')') {
									goto l0
								}
								position++
								depth--
								add(ruleFunctionSelector, position4)
							}
							break
						case '(':
							{
								position8 := position
								depth++
								if buffer[position] != rune('(') {
									goto l0
								}
								position++
								if !_rules[ruleSimpleSelector]() {
									goto l0
								}
							l9:
								{
									position10, tokenIndex10, depth10 := position, tokenIndex, depth
									if buffer[position] != rune(',') {
										goto l10
									}
									position++
									if !_rules[ruleWSX]() {
										goto l10
									}
									if !_rules[ruleSimpleSelector]() {
										goto l10
									}
									goto l9
								l10:
									position, tokenIndex, depth = position10, tokenIndex10, depth10
								}
								if buffer[position] != rune(')') {
									goto l0
								}
								position++
								depth--
								add(ruleCompoundSelector, position8)
							}
							break
						default:
							if !_rules[ruleSimpleSelector]() {
								goto l0
							}
							break
						}
					}

					depth--
					add(ruleSelector, position2)
				}
				if !_rules[ruleWS]() {
					goto l0
				}
				{
					position11 := position
					depth++
					if buffer[position] != rune('F') {
						goto l0
					}
					position++
					if buffer[position] != rune('R') {
						goto l0
					}
					position++
					if buffer[position] != rune('O') {
						goto l0
					}
					position++
					if buffer[position] != rune('M') {
						goto l0
					}
					position++
					if !_rules[ruleWS]() {
						goto l0
					}
					{
						position12 := position
						depth++
						{
							position13, tokenIndex13, depth13 := position, tokenIndex, depth
							{
								position15 := position
								depth++
								if !_rules[ruleNamespacePart]() {
									goto l14
								}
							l16:
								{
									position17, tokenIndex17, depth17 := position, tokenIndex, depth
									if buffer[position] != rune('.') {
										goto l17
									}
									position++
									if !_rules[ruleNamespacePart]() {
										goto l17
									}
									goto l16
								l17:
									position, tokenIndex, depth = position17, tokenIndex17, depth17
								}
								{
									position18, tokenIndex18, depth18 := position, tokenIndex, depth
									if buffer[position] != rune('.') {
										goto l18
									}
									position++
									if !_rules[ruleWildcard]() {
										goto l18
									}
									goto l19
								l18:
									position, tokenIndex, depth = position18, tokenIndex18, depth18
								}
							l19:
								depth--
								add(rulePegText, position15)
							}
							goto l13
						l14:
							position, tokenIndex, depth = position13, tokenIndex13, depth13
							{
								position20 := position
								depth++
								if !_rules[ruleWildcard]() {
									goto l0
								}
								depth--
								add(rulePegText, position20)
							}
						}
					l13:
						depth--
						add(ruleNamespace, position12)
					}
					depth--
					add(ruleSource, position11)
				}
				{
					position21, tokenIndex21, depth21 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l21
					}
					{
						position23 := position
						depth++
						if buffer[position] != rune('W') {
							goto l21
						}
						position++
						if buffer[position] != rune('H') {
							goto l21
						}
						position++
						if buffer[position] != rune('E') {
							goto l21
						}
						position++
						if buffer[position] != rune('R') {
							goto l21
						}
						position++
						if buffer[position] != rune('E') {
							goto l21
						}
						position++
						if !_rules[ruleWS]() {
							goto l21
						}
						if !_rules[ruleMultiCriteria]() {
							goto l21
						}
						depth--
						add(ruleCriteria, position23)
					}
					goto l22
				l21:
					position, tokenIndex, depth = position21, tokenIndex21, depth21
				}
			l22:
				{
					position24, tokenIndex24, depth24 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l24
					}
					{
						position26 := position
						depth++
						if buffer[position] != rune('L') {
							goto l24
						}
						position++
						if buffer[position] != rune('I') {
							goto l24
						}
						position++
						if buffer[position] != rune('M') {
							goto l24
						}
						position++
						if buffer[position] != rune('I') {
							goto l24
						}
						position++
						if buffer[position] != rune('T') {
							goto l24
						}
						position++
						if !_rules[ruleWS]() {
							goto l24
						}
						if !_rules[ruleUInt]() {
							goto l24
						}
						depth--
						add(ruleLimit, position26)
					}
					goto l25
				l24:
					position, tokenIndex, depth = position24, tokenIndex24, depth24
				}
			l25:
				if !_rules[ruleWSX]() {
					goto l0
				}
				{
					position27 := position
					depth++
					{
						position28, tokenIndex28, depth28 := position, tokenIndex, depth
						if !matchDot() {
							goto l28
						}
						goto l0
					l28:
						position, tokenIndex, depth = position28, tokenIndex28, depth28
					}
					depth--
					add(ruleEOF, position27)
				}
				depth--
				add(ruleGrammar, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 Selector <- <((&('C') FunctionSelector) | (&('(') CompoundSelector) | (&('*' | 'b' | 'i' | 'n' | 'p' | 's' | 't') SimpleSelector))> */
		nil,
		/* 2 SimpleSelector <- <<SimpleSelectorOp>> */
		func() bool {
			position30, tokenIndex30, depth30 := position, tokenIndex, depth
			{
				position31 := position
				depth++
				{
					position32 := position
					depth++
					{
						position33 := position
						depth++
						{
							switch buffer[position] {
							case 't':
								if buffer[position] != rune('t') {
									goto l30
								}
								position++
								if buffer[position] != rune('i') {
									goto l30
								}
								position++
								if buffer[position] != rune('m') {
									goto l30
								}
								position++
								if buffer[position] != rune('e') {
									goto l30
								}
								position++
								if buffer[position] != rune('s') {
									goto l30
								}
								position++
								if buffer[position] != rune('t') {
									goto l30
								}
								position++
								if buffer[position] != rune('a') {
									goto l30
								}
								position++
								if buffer[position] != rune('m') {
									goto l30
								}
								position++
								if buffer[position] != rune('p') {
									goto l30
								}
								position++
								break
							case 's':
								if buffer[position] != rune('s') {
									goto l30
								}
								position++
								if buffer[position] != rune('o') {
									goto l30
								}
								position++
								if buffer[position] != rune('u') {
									goto l30
								}
								position++
								if buffer[position] != rune('r') {
									goto l30
								}
								position++
								if buffer[position] != rune('c') {
									goto l30
								}
								position++
								if buffer[position] != rune('e') {
									goto l30
								}
								position++
								break
							case 'n':
								if buffer[position] != rune('n') {
									goto l30
								}
								position++
								if buffer[position] != rune('a') {
									goto l30
								}
								position++
								if buffer[position] != rune('m') {
									goto l30
								}
								position++
								if buffer[position] != rune('e') {
									goto l30
								}
								position++
								if buffer[position] != rune('s') {
									goto l30
								}
								position++
								if buffer[position] != rune('p') {
									goto l30
								}
								position++
								if buffer[position] != rune('a') {
									goto l30
								}
								position++
								if buffer[position] != rune('c') {
									goto l30
								}
								position++
								if buffer[position] != rune('e') {
									goto l30
								}
								position++
								break
							case 'p':
								if buffer[position] != rune('p') {
									goto l30
								}
								position++
								if buffer[position] != rune('u') {
									goto l30
								}
								position++
								if buffer[position] != rune('b') {
									goto l30
								}
								position++
								if buffer[position] != rune('l') {
									goto l30
								}
								position++
								if buffer[position] != rune('i') {
									goto l30
								}
								position++
								if buffer[position] != rune('s') {
									goto l30
								}
								position++
								if buffer[position] != rune('h') {
									goto l30
								}
								position++
								if buffer[position] != rune('e') {
									goto l30
								}
								position++
								if buffer[position] != rune('r') {
									goto l30
								}
								position++
								break
							case 'i':
								if buffer[position] != rune('i') {
									goto l30
								}
								position++
								if buffer[position] != rune('d') {
									goto l30
								}
								position++
								break
							case 'b':
								if buffer[position] != rune('b') {
									goto l30
								}
								position++
								if buffer[position] != rune('o') {
									goto l30
								}
								position++
								if buffer[position] != rune('d') {
									goto l30
								}
								position++
								if buffer[position] != rune('y') {
									goto l30
								}
								position++
								break
							default:
								if buffer[position] != rune('*') {
									goto l30
								}
								position++
								break
							}
						}

						depth--
						add(ruleSimpleSelectorOp, position33)
					}
					depth--
					add(rulePegText, position32)
				}
				depth--
				add(ruleSimpleSelector, position31)
			}
			return true
		l30:
			position, tokenIndex, depth = position30, tokenIndex30, depth30
			return false
		},
		/* 3 SimpleSelectorOp <- <((&('t') ('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p')) | (&('s') ('s' 'o' 'u' 'r' 'c' 'e')) | (&('n') ('n' 'a' 'm' 'e' 's' 'p' 'a' 'c' 'e')) | (&('p') ('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')) | (&('i') ('i' 'd')) | (&('b') ('b' 'o' 'd' 'y')) | (&('*') '*'))> */
		nil,
		/* 4 CompoundSelector <- <('(' SimpleSelector (',' WSX SimpleSelector)* ')')> */
		nil,
		/* 5 FunctionSelector <- <(Function '(' SimpleSelector ')')> */
		nil,
		/* 6 Function <- <<FunctionOp>> */
		nil,
		/* 7 FunctionOp <- <('C' 'O' 'U' 'N' 'T')> */
		nil,
		/* 8 Source <- <('F' 'R' 'O' 'M' WS Namespace)> */
		nil,
		/* 9 Namespace <- <(<(NamespacePart ('.' NamespacePart)* ('.' Wildcard)?)> / <Wildcard>)> */
		nil,
		/* 10 NamespacePart <- <((&('-') '-') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position42, tokenIndex42, depth42 := position, tokenIndex, depth
			{
				position43 := position
				depth++
				{
					switch buffer[position] {
					case '-':
						if buffer[position] != rune('-') {
							goto l42
						}
						position++
						break
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l42
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l42
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l42
						}
						position++
						break
					}
				}

			l44:
				{
					position45, tokenIndex45, depth45 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '-':
							if buffer[position] != rune('-') {
								goto l45
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l45
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l45
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l45
							}
							position++
							break
						}
					}

					goto l44
				l45:
					position, tokenIndex, depth = position45, tokenIndex45, depth45
				}
				depth--
				add(ruleNamespacePart, position43)
			}
			return true
		l42:
			position, tokenIndex, depth = position42, tokenIndex42, depth42
			return false
		},
		/* 11 Wildcard <- <'*'> */
		func() bool {
			position48, tokenIndex48, depth48 := position, tokenIndex, depth
			{
				position49 := position
				depth++
				if buffer[position] != rune('*') {
					goto l48
				}
				position++
				depth--
				add(ruleWildcard, position49)
			}
			return true
		l48:
			position, tokenIndex, depth = position48, tokenIndex48, depth48
			return false
		},
		/* 12 Criteria <- <('W' 'H' 'E' 'R' 'E' WS MultiCriteria)> */
		nil,
		/* 13 MultiCriteria <- <(CompoundCriteria (WS Boolean WS CompoundCriteria)*)> */
		func() bool {
			position51, tokenIndex51, depth51 := position, tokenIndex, depth
			{
				position52 := position
				depth++
				if !_rules[ruleCompoundCriteria]() {
					goto l51
				}
			l53:
				{
					position54, tokenIndex54, depth54 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l54
					}
					{
						position55 := position
						depth++
						{
							position56 := position
							depth++
							{
								position57 := position
								depth++
								{
									position58, tokenIndex58, depth58 := position, tokenIndex, depth
									if buffer[position] != rune('A') {
										goto l59
									}
									position++
									if buffer[position] != rune('N') {
										goto l59
									}
									position++
									if buffer[position] != rune('D') {
										goto l59
									}
									position++
									goto l58
								l59:
									position, tokenIndex, depth = position58, tokenIndex58, depth58
									if buffer[position] != rune('O') {
										goto l54
									}
									position++
									if buffer[position] != rune('R') {
										goto l54
									}
									position++
								}
							l58:
								depth--
								add(ruleBooleanOp, position57)
							}
							depth--
							add(rulePegText, position56)
						}
						depth--
						add(ruleBoolean, position55)
					}
					if !_rules[ruleWS]() {
						goto l54
					}
					if !_rules[ruleCompoundCriteria]() {
						goto l54
					}
					goto l53
				l54:
					position, tokenIndex, depth = position54, tokenIndex54, depth54
				}
				depth--
				add(ruleMultiCriteria, position52)
			}
			return true
		l51:
			position, tokenIndex, depth = position51, tokenIndex51, depth51
			return false
		},
		/* 14 CompoundCriteria <- <(SimpleCriteria / ('(' MultiCriteria ')'))> */
		func() bool {
			position60, tokenIndex60, depth60 := position, tokenIndex, depth
			{
				position61 := position
				depth++
				{
					position62, tokenIndex62, depth62 := position, tokenIndex, depth
					{
						position64 := position
						depth++
						{
							switch buffer[position] {
							case 't':
								{
									position66 := position
									depth++
									if buffer[position] != rune('t') {
										goto l63
									}
									position++
									if buffer[position] != rune('i') {
										goto l63
									}
									position++
									if buffer[position] != rune('m') {
										goto l63
									}
									position++
									if buffer[position] != rune('e') {
										goto l63
									}
									position++
									if buffer[position] != rune('s') {
										goto l63
									}
									position++
									if buffer[position] != rune('t') {
										goto l63
									}
									position++
									if buffer[position] != rune('a') {
										goto l63
									}
									position++
									if buffer[position] != rune('m') {
										goto l63
									}
									position++
									if buffer[position] != rune('p') {
										goto l63
									}
									position++
									if !_rules[ruleWSX]() {
										goto l63
									}
									{
										position67 := position
										depth++
										{
											position68 := position
											depth++
											{
												position69 := position
												depth++
												{
													position70, tokenIndex70, depth70 := position, tokenIndex, depth
													if buffer[position] != rune('<') {
														goto l71
													}
													position++
													if buffer[position] != rune('=') {
														goto l71
													}
													position++
													goto l70
												l71:
													position, tokenIndex, depth = position70, tokenIndex70, depth70
													if buffer[position] != rune('>') {
														goto l72
													}
													position++
													if buffer[position] != rune('=') {
														goto l72
													}
													position++
													goto l70
												l72:
													position, tokenIndex, depth = position70, tokenIndex70, depth70
													{
														switch buffer[position] {
														case '>':
															if buffer[position] != rune('>') {
																goto l63
															}
															position++
															break
														case '=':
															if buffer[position] != rune('=') {
																goto l63
															}
															position++
															break
														default:
															if buffer[position] != rune('<') {
																goto l63
															}
															position++
															break
														}
													}

												}
											l70:
												depth--
												add(ruleComparisonOp, position69)
											}
											depth--
											add(rulePegText, position68)
										}
										depth--
										add(ruleComparison, position67)
									}
									if !_rules[ruleWSX]() {
										goto l63
									}
									if !_rules[ruleUInt]() {
										goto l63
									}
									depth--
									add(ruleTimeCriteria, position66)
								}
								break
							case 's':
								{
									position74 := position
									depth++
									if buffer[position] != rune('s') {
										goto l63
									}
									position++
									if buffer[position] != rune('o') {
										goto l63
									}
									position++
									if buffer[position] != rune('u') {
										goto l63
									}
									position++
									if buffer[position] != rune('r') {
										goto l63
									}
									position++
									if buffer[position] != rune('c') {
										goto l63
									}
									position++
									if buffer[position] != rune('e') {
										goto l63
									}
									position++
									if !_rules[ruleWSX]() {
										goto l63
									}
									if buffer[position] != rune('=') {
										goto l63
									}
									position++
									if !_rules[ruleWSX]() {
										goto l63
									}
									if !_rules[rulePublisherId]() {
										goto l63
									}
									depth--
									add(ruleSourceCriteria, position74)
								}
								break
							case 'p':
								{
									position75 := position
									depth++
									if buffer[position] != rune('p') {
										goto l63
									}
									position++
									if buffer[position] != rune('u') {
										goto l63
									}
									position++
									if buffer[position] != rune('b') {
										goto l63
									}
									position++
									if buffer[position] != rune('l') {
										goto l63
									}
									position++
									if buffer[position] != rune('i') {
										goto l63
									}
									position++
									if buffer[position] != rune('s') {
										goto l63
									}
									position++
									if buffer[position] != rune('h') {
										goto l63
									}
									position++
									if buffer[position] != rune('e') {
										goto l63
									}
									position++
									if buffer[position] != rune('r') {
										goto l63
									}
									position++
									if !_rules[ruleWSX]() {
										goto l63
									}
									if buffer[position] != rune('=') {
										goto l63
									}
									position++
									if !_rules[ruleWSX]() {
										goto l63
									}
									if !_rules[rulePublisherId]() {
										goto l63
									}
									depth--
									add(rulePublisherCriteria, position75)
								}
								break
							default:
								{
									position76 := position
									depth++
									if buffer[position] != rune('i') {
										goto l63
									}
									position++
									if buffer[position] != rune('d') {
										goto l63
									}
									position++
									if !_rules[ruleWSX]() {
										goto l63
									}
									if buffer[position] != rune('=') {
										goto l63
									}
									position++
									if !_rules[ruleWSX]() {
										goto l63
									}
									{
										position77 := position
										depth++
										{
											position78 := position
											depth++
											{
												switch buffer[position] {
												case ':':
													if buffer[position] != rune(':') {
														goto l63
													}
													position++
													break
												case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
													if c := buffer[position]; c < rune('0') || c > rune('9') {
														goto l63
													}
													position++
													break
												case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
													if c := buffer[position]; c < rune('A') || c > rune('Z') {
														goto l63
													}
													position++
													break
												default:
													if c := buffer[position]; c < rune('a') || c > rune('z') {
														goto l63
													}
													position++
													break
												}
											}

										l79:
											{
												position80, tokenIndex80, depth80 := position, tokenIndex, depth
												{
													switch buffer[position] {
													case ':':
														if buffer[position] != rune(':') {
															goto l80
														}
														position++
														break
													case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
														if c := buffer[position]; c < rune('0') || c > rune('9') {
															goto l80
														}
														position++
														break
													case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
														if c := buffer[position]; c < rune('A') || c > rune('Z') {
															goto l80
														}
														position++
														break
													default:
														if c := buffer[position]; c < rune('a') || c > rune('z') {
															goto l80
														}
														position++
														break
													}
												}

												goto l79
											l80:
												position, tokenIndex, depth = position80, tokenIndex80, depth80
											}
											depth--
											add(rulePegText, position78)
										}
										depth--
										add(ruleStatementId, position77)
									}
									depth--
									add(ruleIdCriteria, position76)
								}
								break
							}
						}

						depth--
						add(ruleSimpleCriteria, position64)
					}
					goto l62
				l63:
					position, tokenIndex, depth = position62, tokenIndex62, depth62
					if buffer[position] != rune('(') {
						goto l60
					}
					position++
					if !_rules[ruleMultiCriteria]() {
						goto l60
					}
					if buffer[position] != rune(')') {
						goto l60
					}
					position++
				}
			l62:
				depth--
				add(ruleCompoundCriteria, position61)
			}
			return true
		l60:
			position, tokenIndex, depth = position60, tokenIndex60, depth60
			return false
		},
		/* 15 SimpleCriteria <- <((&('t') TimeCriteria) | (&('s') SourceCriteria) | (&('p') PublisherCriteria) | (&('i') IdCriteria))> */
		nil,
		/* 16 IdCriteria <- <('i' 'd' WSX '=' WSX StatementId)> */
		nil,
		/* 17 PublisherCriteria <- <('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r' WSX '=' WSX PublisherId)> */
		nil,
		/* 18 SourceCriteria <- <('s' 'o' 'u' 'r' 'c' 'e' WSX '=' WSX PublisherId)> */
		nil,
		/* 19 TimeCriteria <- <('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p' WSX Comparison WSX UInt)> */
		nil,
		/* 20 Boolean <- <<BooleanOp>> */
		nil,
		/* 21 BooleanOp <- <(('A' 'N' 'D') / ('O' 'R'))> */
		nil,
		/* 22 StatementId <- <<((&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 23 PublisherId <- <<((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		func() bool {
			position91, tokenIndex91, depth91 := position, tokenIndex, depth
			{
				position92 := position
				depth++
				{
					position93 := position
					depth++
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l91
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l91
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l91
							}
							position++
							break
						}
					}

				l94:
					{
						position95, tokenIndex95, depth95 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l95
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l95
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l95
								}
								position++
								break
							}
						}

						goto l94
					l95:
						position, tokenIndex, depth = position95, tokenIndex95, depth95
					}
					depth--
					add(rulePegText, position93)
				}
				depth--
				add(rulePublisherId, position92)
			}
			return true
		l91:
			position, tokenIndex, depth = position91, tokenIndex91, depth91
			return false
		},
		/* 24 Comparison <- <<ComparisonOp>> */
		nil,
		/* 25 ComparisonOp <- <(('<' '=') / ('>' '=') / ((&('>') '>') | (&('=') '=') | (&('<') '<')))> */
		nil,
		/* 26 Limit <- <('L' 'I' 'M' 'I' 'T' WS UInt)> */
		nil,
		/* 27 UInt <- <<[0-9]+>> */
		func() bool {
			position101, tokenIndex101, depth101 := position, tokenIndex, depth
			{
				position102 := position
				depth++
				{
					position103 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l101
					}
					position++
				l104:
					{
						position105, tokenIndex105, depth105 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l105
						}
						position++
						goto l104
					l105:
						position, tokenIndex, depth = position105, tokenIndex105, depth105
					}
					depth--
					add(rulePegText, position103)
				}
				depth--
				add(ruleUInt, position102)
			}
			return true
		l101:
			position, tokenIndex, depth = position101, tokenIndex101, depth101
			return false
		},
		/* 28 WS <- <WhiteSpace+> */
		func() bool {
			position106, tokenIndex106, depth106 := position, tokenIndex, depth
			{
				position107 := position
				depth++
				if !_rules[ruleWhiteSpace]() {
					goto l106
				}
			l108:
				{
					position109, tokenIndex109, depth109 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l109
					}
					goto l108
				l109:
					position, tokenIndex, depth = position109, tokenIndex109, depth109
				}
				depth--
				add(ruleWS, position107)
			}
			return true
		l106:
			position, tokenIndex, depth = position106, tokenIndex106, depth106
			return false
		},
		/* 29 WSX <- <WhiteSpace*> */
		func() bool {
			{
				position111 := position
				depth++
			l112:
				{
					position113, tokenIndex113, depth113 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l113
					}
					goto l112
				l113:
					position, tokenIndex, depth = position113, tokenIndex113, depth113
				}
				depth--
				add(ruleWSX, position111)
			}
			return true
		},
		/* 30 WhiteSpace <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') EOL))> */
		func() bool {
			position114, tokenIndex114, depth114 := position, tokenIndex, depth
			{
				position115 := position
				depth++
				{
					switch buffer[position] {
					case '\t':
						if buffer[position] != rune('\t') {
							goto l114
						}
						position++
						break
					case ' ':
						if buffer[position] != rune(' ') {
							goto l114
						}
						position++
						break
					default:
						{
							position117 := position
							depth++
							{
								position118, tokenIndex118, depth118 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l119
								}
								position++
								if buffer[position] != rune('\n') {
									goto l119
								}
								position++
								goto l118
							l119:
								position, tokenIndex, depth = position118, tokenIndex118, depth118
								if buffer[position] != rune('\n') {
									goto l120
								}
								position++
								goto l118
							l120:
								position, tokenIndex, depth = position118, tokenIndex118, depth118
								if buffer[position] != rune('\r') {
									goto l114
								}
								position++
							}
						l118:
							depth--
							add(ruleEOL, position117)
						}
						break
					}
				}

				depth--
				add(ruleWhiteSpace, position115)
			}
			return true
		l114:
			position, tokenIndex, depth = position114, tokenIndex114, depth114
			return false
		},
		/* 31 EOL <- <(('\r' '\n') / '\n' / '\r')> */
		nil,
		/* 32 EOF <- <!.> */
		nil,
		nil,
	}
	p.rules = _rules
}
