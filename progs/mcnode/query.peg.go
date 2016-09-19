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
	ruleCompoundSelector
	ruleFunctionSelector
	ruleFunction
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
	"CompoundSelector",
	"FunctionSelector",
	"Function",
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
	Buffer string
	buffer []rune
	rules  [33]func() bool
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
								position6 := position
								depth++
								if buffer[position] != rune('(') {
									goto l0
								}
								position++
								if !_rules[ruleSimpleSelector]() {
									goto l0
								}
							l7:
								{
									position8, tokenIndex8, depth8 := position, tokenIndex, depth
									if buffer[position] != rune(',') {
										goto l8
									}
									position++
									if !_rules[ruleWSX]() {
										goto l8
									}
									if !_rules[ruleSimpleSelector]() {
										goto l8
									}
									goto l7
								l8:
									position, tokenIndex, depth = position8, tokenIndex8, depth8
								}
								if buffer[position] != rune(')') {
									goto l0
								}
								position++
								depth--
								add(ruleCompoundSelector, position6)
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
					position9 := position
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
						position10 := position
						depth++
						{
							position11, tokenIndex11, depth11 := position, tokenIndex, depth
							{
								position13 := position
								depth++
								if !_rules[ruleNamespacePart]() {
									goto l12
								}
							l14:
								{
									position15, tokenIndex15, depth15 := position, tokenIndex, depth
									if buffer[position] != rune('.') {
										goto l15
									}
									position++
									if !_rules[ruleNamespacePart]() {
										goto l15
									}
									goto l14
								l15:
									position, tokenIndex, depth = position15, tokenIndex15, depth15
								}
								{
									position16, tokenIndex16, depth16 := position, tokenIndex, depth
									if buffer[position] != rune('.') {
										goto l16
									}
									position++
									if !_rules[ruleWildcard]() {
										goto l16
									}
									goto l17
								l16:
									position, tokenIndex, depth = position16, tokenIndex16, depth16
								}
							l17:
								depth--
								add(rulePegText, position13)
							}
							goto l11
						l12:
							position, tokenIndex, depth = position11, tokenIndex11, depth11
							{
								position18 := position
								depth++
								if !_rules[ruleWildcard]() {
									goto l0
								}
								depth--
								add(rulePegText, position18)
							}
						}
					l11:
						depth--
						add(ruleNamespace, position10)
					}
					depth--
					add(ruleSource, position9)
				}
				{
					position19, tokenIndex19, depth19 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l19
					}
					{
						position21 := position
						depth++
						if buffer[position] != rune('W') {
							goto l19
						}
						position++
						if buffer[position] != rune('H') {
							goto l19
						}
						position++
						if buffer[position] != rune('E') {
							goto l19
						}
						position++
						if buffer[position] != rune('R') {
							goto l19
						}
						position++
						if buffer[position] != rune('E') {
							goto l19
						}
						position++
						if !_rules[ruleWS]() {
							goto l19
						}
						if !_rules[ruleMultiCriteria]() {
							goto l19
						}
						depth--
						add(ruleCriteria, position21)
					}
					goto l20
				l19:
					position, tokenIndex, depth = position19, tokenIndex19, depth19
				}
			l20:
				{
					position22, tokenIndex22, depth22 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l22
					}
					{
						position24 := position
						depth++
						if buffer[position] != rune('L') {
							goto l22
						}
						position++
						if buffer[position] != rune('I') {
							goto l22
						}
						position++
						if buffer[position] != rune('M') {
							goto l22
						}
						position++
						if buffer[position] != rune('I') {
							goto l22
						}
						position++
						if buffer[position] != rune('T') {
							goto l22
						}
						position++
						if !_rules[ruleWS]() {
							goto l22
						}
						if !_rules[ruleUInt]() {
							goto l22
						}
						depth--
						add(ruleLimit, position24)
					}
					goto l23
				l22:
					position, tokenIndex, depth = position22, tokenIndex22, depth22
				}
			l23:
				if !_rules[ruleWSX]() {
					goto l0
				}
				{
					position25 := position
					depth++
					{
						position26, tokenIndex26, depth26 := position, tokenIndex, depth
						if !matchDot() {
							goto l26
						}
						goto l0
					l26:
						position, tokenIndex, depth = position26, tokenIndex26, depth26
					}
					depth--
					add(ruleEOF, position25)
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
		/* 2 SimpleSelector <- <((&('t') ('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p')) | (&('s') ('s' 'o' 'u' 'r' 'c' 'e')) | (&('n') ('n' 'a' 'm' 'e' 's' 'p' 'a' 'c' 'e')) | (&('p') ('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')) | (&('i') ('i' 'd')) | (&('b') ('b' 'o' 'd' 'y')) | (&('*') '*'))> */
		func() bool {
			position28, tokenIndex28, depth28 := position, tokenIndex, depth
			{
				position29 := position
				depth++
				{
					switch buffer[position] {
					case 't':
						if buffer[position] != rune('t') {
							goto l28
						}
						position++
						if buffer[position] != rune('i') {
							goto l28
						}
						position++
						if buffer[position] != rune('m') {
							goto l28
						}
						position++
						if buffer[position] != rune('e') {
							goto l28
						}
						position++
						if buffer[position] != rune('s') {
							goto l28
						}
						position++
						if buffer[position] != rune('t') {
							goto l28
						}
						position++
						if buffer[position] != rune('a') {
							goto l28
						}
						position++
						if buffer[position] != rune('m') {
							goto l28
						}
						position++
						if buffer[position] != rune('p') {
							goto l28
						}
						position++
						break
					case 's':
						if buffer[position] != rune('s') {
							goto l28
						}
						position++
						if buffer[position] != rune('o') {
							goto l28
						}
						position++
						if buffer[position] != rune('u') {
							goto l28
						}
						position++
						if buffer[position] != rune('r') {
							goto l28
						}
						position++
						if buffer[position] != rune('c') {
							goto l28
						}
						position++
						if buffer[position] != rune('e') {
							goto l28
						}
						position++
						break
					case 'n':
						if buffer[position] != rune('n') {
							goto l28
						}
						position++
						if buffer[position] != rune('a') {
							goto l28
						}
						position++
						if buffer[position] != rune('m') {
							goto l28
						}
						position++
						if buffer[position] != rune('e') {
							goto l28
						}
						position++
						if buffer[position] != rune('s') {
							goto l28
						}
						position++
						if buffer[position] != rune('p') {
							goto l28
						}
						position++
						if buffer[position] != rune('a') {
							goto l28
						}
						position++
						if buffer[position] != rune('c') {
							goto l28
						}
						position++
						if buffer[position] != rune('e') {
							goto l28
						}
						position++
						break
					case 'p':
						if buffer[position] != rune('p') {
							goto l28
						}
						position++
						if buffer[position] != rune('u') {
							goto l28
						}
						position++
						if buffer[position] != rune('b') {
							goto l28
						}
						position++
						if buffer[position] != rune('l') {
							goto l28
						}
						position++
						if buffer[position] != rune('i') {
							goto l28
						}
						position++
						if buffer[position] != rune('s') {
							goto l28
						}
						position++
						if buffer[position] != rune('h') {
							goto l28
						}
						position++
						if buffer[position] != rune('e') {
							goto l28
						}
						position++
						if buffer[position] != rune('r') {
							goto l28
						}
						position++
						break
					case 'i':
						if buffer[position] != rune('i') {
							goto l28
						}
						position++
						if buffer[position] != rune('d') {
							goto l28
						}
						position++
						break
					case 'b':
						if buffer[position] != rune('b') {
							goto l28
						}
						position++
						if buffer[position] != rune('o') {
							goto l28
						}
						position++
						if buffer[position] != rune('d') {
							goto l28
						}
						position++
						if buffer[position] != rune('y') {
							goto l28
						}
						position++
						break
					default:
						if buffer[position] != rune('*') {
							goto l28
						}
						position++
						break
					}
				}

				depth--
				add(ruleSimpleSelector, position29)
			}
			return true
		l28:
			position, tokenIndex, depth = position28, tokenIndex28, depth28
			return false
		},
		/* 3 CompoundSelector <- <('(' SimpleSelector (',' WSX SimpleSelector)* ')')> */
		nil,
		/* 4 FunctionSelector <- <(Function '(' SimpleSelector ')')> */
		nil,
		/* 5 Function <- <('C' 'O' 'U' 'N' 'T')> */
		nil,
		/* 6 Source <- <('F' 'R' 'O' 'M' WS Namespace)> */
		nil,
		/* 7 Namespace <- <(<(NamespacePart ('.' NamespacePart)* ('.' Wildcard)?)> / <Wildcard>)> */
		nil,
		/* 8 NamespacePart <- <((&('-') '-') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position36, tokenIndex36, depth36 := position, tokenIndex, depth
			{
				position37 := position
				depth++
				{
					switch buffer[position] {
					case '-':
						if buffer[position] != rune('-') {
							goto l36
						}
						position++
						break
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l36
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l36
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l36
						}
						position++
						break
					}
				}

			l38:
				{
					position39, tokenIndex39, depth39 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '-':
							if buffer[position] != rune('-') {
								goto l39
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l39
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l39
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l39
							}
							position++
							break
						}
					}

					goto l38
				l39:
					position, tokenIndex, depth = position39, tokenIndex39, depth39
				}
				depth--
				add(ruleNamespacePart, position37)
			}
			return true
		l36:
			position, tokenIndex, depth = position36, tokenIndex36, depth36
			return false
		},
		/* 9 Wildcard <- <'*'> */
		func() bool {
			position42, tokenIndex42, depth42 := position, tokenIndex, depth
			{
				position43 := position
				depth++
				if buffer[position] != rune('*') {
					goto l42
				}
				position++
				depth--
				add(ruleWildcard, position43)
			}
			return true
		l42:
			position, tokenIndex, depth = position42, tokenIndex42, depth42
			return false
		},
		/* 10 Criteria <- <('W' 'H' 'E' 'R' 'E' WS MultiCriteria)> */
		nil,
		/* 11 MultiCriteria <- <(CompoundCriteria (WS Boolean WS CompoundCriteria)*)> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				if !_rules[ruleCompoundCriteria]() {
					goto l45
				}
			l47:
				{
					position48, tokenIndex48, depth48 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l48
					}
					{
						position49 := position
						depth++
						{
							position50 := position
							depth++
							{
								position51 := position
								depth++
								{
									position52, tokenIndex52, depth52 := position, tokenIndex, depth
									if buffer[position] != rune('A') {
										goto l53
									}
									position++
									if buffer[position] != rune('N') {
										goto l53
									}
									position++
									if buffer[position] != rune('D') {
										goto l53
									}
									position++
									goto l52
								l53:
									position, tokenIndex, depth = position52, tokenIndex52, depth52
									if buffer[position] != rune('O') {
										goto l48
									}
									position++
									if buffer[position] != rune('R') {
										goto l48
									}
									position++
								}
							l52:
								depth--
								add(ruleBooleanOp, position51)
							}
							depth--
							add(rulePegText, position50)
						}
						depth--
						add(ruleBoolean, position49)
					}
					if !_rules[ruleWS]() {
						goto l48
					}
					if !_rules[ruleCompoundCriteria]() {
						goto l48
					}
					goto l47
				l48:
					position, tokenIndex, depth = position48, tokenIndex48, depth48
				}
				depth--
				add(ruleMultiCriteria, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 12 CompoundCriteria <- <(SimpleCriteria / ('(' MultiCriteria ')'))> */
		func() bool {
			position54, tokenIndex54, depth54 := position, tokenIndex, depth
			{
				position55 := position
				depth++
				{
					position56, tokenIndex56, depth56 := position, tokenIndex, depth
					{
						position58 := position
						depth++
						{
							switch buffer[position] {
							case 't':
								{
									position60 := position
									depth++
									if buffer[position] != rune('t') {
										goto l57
									}
									position++
									if buffer[position] != rune('i') {
										goto l57
									}
									position++
									if buffer[position] != rune('m') {
										goto l57
									}
									position++
									if buffer[position] != rune('e') {
										goto l57
									}
									position++
									if buffer[position] != rune('s') {
										goto l57
									}
									position++
									if buffer[position] != rune('t') {
										goto l57
									}
									position++
									if buffer[position] != rune('a') {
										goto l57
									}
									position++
									if buffer[position] != rune('m') {
										goto l57
									}
									position++
									if buffer[position] != rune('p') {
										goto l57
									}
									position++
									if !_rules[ruleWSX]() {
										goto l57
									}
									{
										position61 := position
										depth++
										{
											position62 := position
											depth++
											{
												position63 := position
												depth++
												{
													position64, tokenIndex64, depth64 := position, tokenIndex, depth
													if buffer[position] != rune('<') {
														goto l65
													}
													position++
													if buffer[position] != rune('=') {
														goto l65
													}
													position++
													goto l64
												l65:
													position, tokenIndex, depth = position64, tokenIndex64, depth64
													if buffer[position] != rune('>') {
														goto l66
													}
													position++
													if buffer[position] != rune('=') {
														goto l66
													}
													position++
													goto l64
												l66:
													position, tokenIndex, depth = position64, tokenIndex64, depth64
													{
														switch buffer[position] {
														case '>':
															if buffer[position] != rune('>') {
																goto l57
															}
															position++
															break
														case '=':
															if buffer[position] != rune('=') {
																goto l57
															}
															position++
															break
														default:
															if buffer[position] != rune('<') {
																goto l57
															}
															position++
															break
														}
													}

												}
											l64:
												depth--
												add(ruleComparisonOp, position63)
											}
											depth--
											add(rulePegText, position62)
										}
										depth--
										add(ruleComparison, position61)
									}
									if !_rules[ruleWSX]() {
										goto l57
									}
									if !_rules[ruleUInt]() {
										goto l57
									}
									depth--
									add(ruleTimeCriteria, position60)
								}
								break
							case 's':
								{
									position68 := position
									depth++
									if buffer[position] != rune('s') {
										goto l57
									}
									position++
									if buffer[position] != rune('o') {
										goto l57
									}
									position++
									if buffer[position] != rune('u') {
										goto l57
									}
									position++
									if buffer[position] != rune('r') {
										goto l57
									}
									position++
									if buffer[position] != rune('c') {
										goto l57
									}
									position++
									if buffer[position] != rune('e') {
										goto l57
									}
									position++
									if !_rules[ruleWSX]() {
										goto l57
									}
									if buffer[position] != rune('=') {
										goto l57
									}
									position++
									if !_rules[ruleWSX]() {
										goto l57
									}
									if !_rules[rulePublisherId]() {
										goto l57
									}
									depth--
									add(ruleSourceCriteria, position68)
								}
								break
							case 'p':
								{
									position69 := position
									depth++
									if buffer[position] != rune('p') {
										goto l57
									}
									position++
									if buffer[position] != rune('u') {
										goto l57
									}
									position++
									if buffer[position] != rune('b') {
										goto l57
									}
									position++
									if buffer[position] != rune('l') {
										goto l57
									}
									position++
									if buffer[position] != rune('i') {
										goto l57
									}
									position++
									if buffer[position] != rune('s') {
										goto l57
									}
									position++
									if buffer[position] != rune('h') {
										goto l57
									}
									position++
									if buffer[position] != rune('e') {
										goto l57
									}
									position++
									if buffer[position] != rune('r') {
										goto l57
									}
									position++
									if !_rules[ruleWSX]() {
										goto l57
									}
									if buffer[position] != rune('=') {
										goto l57
									}
									position++
									if !_rules[ruleWSX]() {
										goto l57
									}
									if !_rules[rulePublisherId]() {
										goto l57
									}
									depth--
									add(rulePublisherCriteria, position69)
								}
								break
							default:
								{
									position70 := position
									depth++
									if buffer[position] != rune('i') {
										goto l57
									}
									position++
									if buffer[position] != rune('d') {
										goto l57
									}
									position++
									if !_rules[ruleWSX]() {
										goto l57
									}
									if buffer[position] != rune('=') {
										goto l57
									}
									position++
									if !_rules[ruleWSX]() {
										goto l57
									}
									{
										position71 := position
										depth++
										{
											position72 := position
											depth++
											{
												switch buffer[position] {
												case ':':
													if buffer[position] != rune(':') {
														goto l57
													}
													position++
													break
												case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
													if c := buffer[position]; c < rune('0') || c > rune('9') {
														goto l57
													}
													position++
													break
												case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
													if c := buffer[position]; c < rune('A') || c > rune('Z') {
														goto l57
													}
													position++
													break
												default:
													if c := buffer[position]; c < rune('a') || c > rune('z') {
														goto l57
													}
													position++
													break
												}
											}

										l73:
											{
												position74, tokenIndex74, depth74 := position, tokenIndex, depth
												{
													switch buffer[position] {
													case ':':
														if buffer[position] != rune(':') {
															goto l74
														}
														position++
														break
													case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
														if c := buffer[position]; c < rune('0') || c > rune('9') {
															goto l74
														}
														position++
														break
													case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
														if c := buffer[position]; c < rune('A') || c > rune('Z') {
															goto l74
														}
														position++
														break
													default:
														if c := buffer[position]; c < rune('a') || c > rune('z') {
															goto l74
														}
														position++
														break
													}
												}

												goto l73
											l74:
												position, tokenIndex, depth = position74, tokenIndex74, depth74
											}
											depth--
											add(rulePegText, position72)
										}
										depth--
										add(ruleStatementId, position71)
									}
									depth--
									add(ruleIdCriteria, position70)
								}
								break
							}
						}

						depth--
						add(ruleSimpleCriteria, position58)
					}
					goto l56
				l57:
					position, tokenIndex, depth = position56, tokenIndex56, depth56
					if buffer[position] != rune('(') {
						goto l54
					}
					position++
					if !_rules[ruleMultiCriteria]() {
						goto l54
					}
					if buffer[position] != rune(')') {
						goto l54
					}
					position++
				}
			l56:
				depth--
				add(ruleCompoundCriteria, position55)
			}
			return true
		l54:
			position, tokenIndex, depth = position54, tokenIndex54, depth54
			return false
		},
		/* 13 SimpleCriteria <- <((&('t') TimeCriteria) | (&('s') SourceCriteria) | (&('p') PublisherCriteria) | (&('i') IdCriteria))> */
		nil,
		/* 14 IdCriteria <- <('i' 'd' WSX '=' WSX StatementId)> */
		nil,
		/* 15 PublisherCriteria <- <('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r' WSX '=' WSX PublisherId)> */
		nil,
		/* 16 SourceCriteria <- <('s' 'o' 'u' 'r' 'c' 'e' WSX '=' WSX PublisherId)> */
		nil,
		/* 17 TimeCriteria <- <('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p' WSX Comparison WSX UInt)> */
		nil,
		/* 18 Boolean <- <<BooleanOp>> */
		nil,
		/* 19 BooleanOp <- <(('A' 'N' 'D') / ('O' 'R'))> */
		nil,
		/* 20 StatementId <- <<((&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 21 PublisherId <- <<((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		func() bool {
			position85, tokenIndex85, depth85 := position, tokenIndex, depth
			{
				position86 := position
				depth++
				{
					position87 := position
					depth++
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l85
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l85
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l85
							}
							position++
							break
						}
					}

				l88:
					{
						position89, tokenIndex89, depth89 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l89
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l89
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l89
								}
								position++
								break
							}
						}

						goto l88
					l89:
						position, tokenIndex, depth = position89, tokenIndex89, depth89
					}
					depth--
					add(rulePegText, position87)
				}
				depth--
				add(rulePublisherId, position86)
			}
			return true
		l85:
			position, tokenIndex, depth = position85, tokenIndex85, depth85
			return false
		},
		/* 22 Comparison <- <<ComparisonOp>> */
		nil,
		/* 23 ComparisonOp <- <(('<' '=') / ('>' '=') / ((&('>') '>') | (&('=') '=') | (&('<') '<')))> */
		nil,
		/* 24 Limit <- <('L' 'I' 'M' 'I' 'T' WS UInt)> */
		nil,
		/* 25 UInt <- <<[0-9]+>> */
		func() bool {
			position95, tokenIndex95, depth95 := position, tokenIndex, depth
			{
				position96 := position
				depth++
				{
					position97 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l95
					}
					position++
				l98:
					{
						position99, tokenIndex99, depth99 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l99
						}
						position++
						goto l98
					l99:
						position, tokenIndex, depth = position99, tokenIndex99, depth99
					}
					depth--
					add(rulePegText, position97)
				}
				depth--
				add(ruleUInt, position96)
			}
			return true
		l95:
			position, tokenIndex, depth = position95, tokenIndex95, depth95
			return false
		},
		/* 26 WS <- <WhiteSpace+> */
		func() bool {
			position100, tokenIndex100, depth100 := position, tokenIndex, depth
			{
				position101 := position
				depth++
				if !_rules[ruleWhiteSpace]() {
					goto l100
				}
			l102:
				{
					position103, tokenIndex103, depth103 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l103
					}
					goto l102
				l103:
					position, tokenIndex, depth = position103, tokenIndex103, depth103
				}
				depth--
				add(ruleWS, position101)
			}
			return true
		l100:
			position, tokenIndex, depth = position100, tokenIndex100, depth100
			return false
		},
		/* 27 WSX <- <WhiteSpace*> */
		func() bool {
			{
				position105 := position
				depth++
			l106:
				{
					position107, tokenIndex107, depth107 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l107
					}
					goto l106
				l107:
					position, tokenIndex, depth = position107, tokenIndex107, depth107
				}
				depth--
				add(ruleWSX, position105)
			}
			return true
		},
		/* 28 WhiteSpace <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') EOL))> */
		func() bool {
			position108, tokenIndex108, depth108 := position, tokenIndex, depth
			{
				position109 := position
				depth++
				{
					switch buffer[position] {
					case '\t':
						if buffer[position] != rune('\t') {
							goto l108
						}
						position++
						break
					case ' ':
						if buffer[position] != rune(' ') {
							goto l108
						}
						position++
						break
					default:
						{
							position111 := position
							depth++
							{
								position112, tokenIndex112, depth112 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l113
								}
								position++
								if buffer[position] != rune('\n') {
									goto l113
								}
								position++
								goto l112
							l113:
								position, tokenIndex, depth = position112, tokenIndex112, depth112
								if buffer[position] != rune('\n') {
									goto l114
								}
								position++
								goto l112
							l114:
								position, tokenIndex, depth = position112, tokenIndex112, depth112
								if buffer[position] != rune('\r') {
									goto l108
								}
								position++
							}
						l112:
							depth--
							add(ruleEOL, position111)
						}
						break
					}
				}

				depth--
				add(ruleWhiteSpace, position109)
			}
			return true
		l108:
			position, tokenIndex, depth = position108, tokenIndex108, depth108
			return false
		},
		/* 29 EOL <- <(('\r' '\n') / '\n' / '\r')> */
		nil,
		/* 30 EOF <- <!.> */
		nil,
		nil,
	}
	p.rules = _rules
}
