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
	ruleSimpleNamespace
	ruleNamespacePart
	ruleWildcardNamespace
	ruleCriteria
	ruleCriteriaSpec
	ruleSimpleCriteria
	ruleMultiCriteria
	ruleIdCriteria
	rulePublisherCriteria
	ruleSourceCriteria
	ruleTimeCriteria
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
	"SimpleNamespace",
	"NamespacePart",
	"WildcardNamespace",
	"Criteria",
	"CriteriaSpec",
	"SimpleCriteria",
	"MultiCriteria",
	"IdCriteria",
	"PublisherCriteria",
	"SourceCriteria",
	"TimeCriteria",
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
	rules  [32]func() bool
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
							if !_rules[ruleSimpleNamespace]() {
								goto l12
							}
							goto l11
						l12:
							position, tokenIndex, depth = position11, tokenIndex11, depth11
							{
								position13 := position
								depth++
								{
									position14 := position
									depth++
									if !_rules[ruleSimpleNamespace]() {
										goto l0
									}
									if buffer[position] != rune('.') {
										goto l0
									}
									position++
									if buffer[position] != rune('*') {
										goto l0
									}
									position++
									depth--
									add(rulePegText, position14)
								}
								depth--
								add(ruleWildcardNamespace, position13)
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
					position15, tokenIndex15, depth15 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l15
					}
					{
						position17 := position
						depth++
						if buffer[position] != rune('W') {
							goto l15
						}
						position++
						if buffer[position] != rune('H') {
							goto l15
						}
						position++
						if buffer[position] != rune('E') {
							goto l15
						}
						position++
						if buffer[position] != rune('R') {
							goto l15
						}
						position++
						if buffer[position] != rune('E') {
							goto l15
						}
						position++
						if !_rules[ruleWS]() {
							goto l15
						}
						{
							position18 := position
							depth++
							{
								position19, tokenIndex19, depth19 := position, tokenIndex, depth
								if !_rules[ruleSimpleCriteria]() {
									goto l20
								}
								goto l19
							l20:
								position, tokenIndex, depth = position19, tokenIndex19, depth19
								{
									position21 := position
									depth++
									if !_rules[ruleSimpleCriteria]() {
										goto l15
									}
								l22:
									{
										position23, tokenIndex23, depth23 := position, tokenIndex, depth
										if !_rules[ruleWS]() {
											goto l23
										}
										if buffer[position] != rune('A') {
											goto l23
										}
										position++
										if buffer[position] != rune('N') {
											goto l23
										}
										position++
										if buffer[position] != rune('D') {
											goto l23
										}
										position++
										if !_rules[ruleWS]() {
											goto l23
										}
										if !_rules[ruleSimpleCriteria]() {
											goto l23
										}
										goto l22
									l23:
										position, tokenIndex, depth = position23, tokenIndex23, depth23
									}
									depth--
									add(ruleMultiCriteria, position21)
								}
							}
						l19:
							depth--
							add(ruleCriteriaSpec, position18)
						}
						depth--
						add(ruleCriteria, position17)
					}
					goto l16
				l15:
					position, tokenIndex, depth = position15, tokenIndex15, depth15
				}
			l16:
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
		/* 1 Selector <- <((&('C') FunctionSelector) | (&('(') CompoundSelector) | (&('*' | 'b' | 'i' | 'p' | 's' | 't') SimpleSelector))> */
		nil,
		/* 2 SimpleSelector <- <((&('t') ('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p')) | (&('s') ('s' 'o' 'u' 'r' 'c' 'e')) | (&('p') ('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')) | (&('i') ('i' 'd')) | (&('b') ('b' 'o' 'd' 'y')) | (&('*') '*'))> */
		func() bool {
			position30, tokenIndex30, depth30 := position, tokenIndex, depth
			{
				position31 := position
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
				add(ruleSimpleSelector, position31)
			}
			return true
		l30:
			position, tokenIndex, depth = position30, tokenIndex30, depth30
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
		/* 7 Namespace <- <(SimpleNamespace / WildcardNamespace)> */
		nil,
		/* 8 SimpleNamespace <- <<(NamespacePart ('.' NamespacePart)*)>> */
		func() bool {
			position38, tokenIndex38, depth38 := position, tokenIndex, depth
			{
				position39 := position
				depth++
				{
					position40 := position
					depth++
					if !_rules[ruleNamespacePart]() {
						goto l38
					}
				l41:
					{
						position42, tokenIndex42, depth42 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l42
						}
						position++
						if !_rules[ruleNamespacePart]() {
							goto l42
						}
						goto l41
					l42:
						position, tokenIndex, depth = position42, tokenIndex42, depth42
					}
					depth--
					add(rulePegText, position40)
				}
				depth--
				add(ruleSimpleNamespace, position39)
			}
			return true
		l38:
			position, tokenIndex, depth = position38, tokenIndex38, depth38
			return false
		},
		/* 9 NamespacePart <- <((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position43, tokenIndex43, depth43 := position, tokenIndex, depth
			{
				position44 := position
				depth++
				{
					switch buffer[position] {
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l43
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l43
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l43
						}
						position++
						break
					}
				}

			l45:
				{
					position46, tokenIndex46, depth46 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l46
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l46
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l46
							}
							position++
							break
						}
					}

					goto l45
				l46:
					position, tokenIndex, depth = position46, tokenIndex46, depth46
				}
				depth--
				add(ruleNamespacePart, position44)
			}
			return true
		l43:
			position, tokenIndex, depth = position43, tokenIndex43, depth43
			return false
		},
		/* 10 WildcardNamespace <- <<(SimpleNamespace ('.' '*'))>> */
		nil,
		/* 11 Criteria <- <('W' 'H' 'E' 'R' 'E' WS CriteriaSpec)> */
		nil,
		/* 12 CriteriaSpec <- <(SimpleCriteria / MultiCriteria)> */
		nil,
		/* 13 SimpleCriteria <- <((&('t') TimeCriteria) | (&('s') SourceCriteria) | (&('p') PublisherCriteria) | (&('i') IdCriteria))> */
		func() bool {
			position52, tokenIndex52, depth52 := position, tokenIndex, depth
			{
				position53 := position
				depth++
				{
					switch buffer[position] {
					case 't':
						{
							position55 := position
							depth++
							if buffer[position] != rune('t') {
								goto l52
							}
							position++
							if buffer[position] != rune('i') {
								goto l52
							}
							position++
							if buffer[position] != rune('m') {
								goto l52
							}
							position++
							if buffer[position] != rune('e') {
								goto l52
							}
							position++
							if buffer[position] != rune('s') {
								goto l52
							}
							position++
							if buffer[position] != rune('t') {
								goto l52
							}
							position++
							if buffer[position] != rune('a') {
								goto l52
							}
							position++
							if buffer[position] != rune('m') {
								goto l52
							}
							position++
							if buffer[position] != rune('p') {
								goto l52
							}
							position++
							if !_rules[ruleWSX]() {
								goto l52
							}
							{
								position56 := position
								depth++
								{
									position57 := position
									depth++
									{
										position58 := position
										depth++
										{
											position59, tokenIndex59, depth59 := position, tokenIndex, depth
											if buffer[position] != rune('<') {
												goto l60
											}
											position++
											goto l59
										l60:
											position, tokenIndex, depth = position59, tokenIndex59, depth59
											if buffer[position] != rune('>') {
												goto l61
											}
											position++
											if buffer[position] != rune('=') {
												goto l61
											}
											position++
											goto l59
										l61:
											position, tokenIndex, depth = position59, tokenIndex59, depth59
											{
												switch buffer[position] {
												case '>':
													if buffer[position] != rune('>') {
														goto l52
													}
													position++
													break
												case '=':
													if buffer[position] != rune('=') {
														goto l52
													}
													position++
													break
												default:
													if buffer[position] != rune('<') {
														goto l52
													}
													position++
													if buffer[position] != rune('=') {
														goto l52
													}
													position++
													break
												}
											}

										}
									l59:
										depth--
										add(ruleComparisonOp, position58)
									}
									depth--
									add(rulePegText, position57)
								}
								depth--
								add(ruleComparison, position56)
							}
							if !_rules[ruleWSX]() {
								goto l52
							}
							if !_rules[ruleUInt]() {
								goto l52
							}
							depth--
							add(ruleTimeCriteria, position55)
						}
						break
					case 's':
						{
							position63 := position
							depth++
							if buffer[position] != rune('s') {
								goto l52
							}
							position++
							if buffer[position] != rune('o') {
								goto l52
							}
							position++
							if buffer[position] != rune('u') {
								goto l52
							}
							position++
							if buffer[position] != rune('r') {
								goto l52
							}
							position++
							if buffer[position] != rune('c') {
								goto l52
							}
							position++
							if buffer[position] != rune('e') {
								goto l52
							}
							position++
							if !_rules[ruleWSX]() {
								goto l52
							}
							if buffer[position] != rune('=') {
								goto l52
							}
							position++
							if !_rules[ruleWSX]() {
								goto l52
							}
							if !_rules[rulePublisherId]() {
								goto l52
							}
							depth--
							add(ruleSourceCriteria, position63)
						}
						break
					case 'p':
						{
							position64 := position
							depth++
							if buffer[position] != rune('p') {
								goto l52
							}
							position++
							if buffer[position] != rune('u') {
								goto l52
							}
							position++
							if buffer[position] != rune('b') {
								goto l52
							}
							position++
							if buffer[position] != rune('l') {
								goto l52
							}
							position++
							if buffer[position] != rune('i') {
								goto l52
							}
							position++
							if buffer[position] != rune('s') {
								goto l52
							}
							position++
							if buffer[position] != rune('h') {
								goto l52
							}
							position++
							if buffer[position] != rune('e') {
								goto l52
							}
							position++
							if buffer[position] != rune('r') {
								goto l52
							}
							position++
							if !_rules[ruleWSX]() {
								goto l52
							}
							if buffer[position] != rune('=') {
								goto l52
							}
							position++
							if !_rules[ruleWSX]() {
								goto l52
							}
							if !_rules[rulePublisherId]() {
								goto l52
							}
							depth--
							add(rulePublisherCriteria, position64)
						}
						break
					default:
						{
							position65 := position
							depth++
							if buffer[position] != rune('i') {
								goto l52
							}
							position++
							if buffer[position] != rune('d') {
								goto l52
							}
							position++
							if !_rules[ruleWSX]() {
								goto l52
							}
							if buffer[position] != rune('=') {
								goto l52
							}
							position++
							if !_rules[ruleWSX]() {
								goto l52
							}
							{
								position66 := position
								depth++
								{
									position67 := position
									depth++
									{
										switch buffer[position] {
										case ':':
											if buffer[position] != rune(':') {
												goto l52
											}
											position++
											break
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l52
											}
											position++
											break
										case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l52
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('a') || c > rune('z') {
												goto l52
											}
											position++
											break
										}
									}

								l68:
									{
										position69, tokenIndex69, depth69 := position, tokenIndex, depth
										{
											switch buffer[position] {
											case ':':
												if buffer[position] != rune(':') {
													goto l69
												}
												position++
												break
											case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
												if c := buffer[position]; c < rune('0') || c > rune('9') {
													goto l69
												}
												position++
												break
											case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
												if c := buffer[position]; c < rune('A') || c > rune('Z') {
													goto l69
												}
												position++
												break
											default:
												if c := buffer[position]; c < rune('a') || c > rune('z') {
													goto l69
												}
												position++
												break
											}
										}

										goto l68
									l69:
										position, tokenIndex, depth = position69, tokenIndex69, depth69
									}
									depth--
									add(rulePegText, position67)
								}
								depth--
								add(ruleStatementId, position66)
							}
							depth--
							add(ruleIdCriteria, position65)
						}
						break
					}
				}

				depth--
				add(ruleSimpleCriteria, position53)
			}
			return true
		l52:
			position, tokenIndex, depth = position52, tokenIndex52, depth52
			return false
		},
		/* 14 MultiCriteria <- <(SimpleCriteria (WS ('A' 'N' 'D') WS SimpleCriteria)*)> */
		nil,
		/* 15 IdCriteria <- <('i' 'd' WSX '=' WSX StatementId)> */
		nil,
		/* 16 PublisherCriteria <- <('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r' WSX '=' WSX PublisherId)> */
		nil,
		/* 17 SourceCriteria <- <('s' 'o' 'u' 'r' 'c' 'e' WSX '=' WSX PublisherId)> */
		nil,
		/* 18 TimeCriteria <- <('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p' WSX Comparison WSX UInt)> */
		nil,
		/* 19 StatementId <- <<((&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 20 PublisherId <- <<((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		func() bool {
			position78, tokenIndex78, depth78 := position, tokenIndex, depth
			{
				position79 := position
				depth++
				{
					position80 := position
					depth++
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l78
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l78
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l78
							}
							position++
							break
						}
					}

				l81:
					{
						position82, tokenIndex82, depth82 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l82
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l82
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l82
								}
								position++
								break
							}
						}

						goto l81
					l82:
						position, tokenIndex, depth = position82, tokenIndex82, depth82
					}
					depth--
					add(rulePegText, position80)
				}
				depth--
				add(rulePublisherId, position79)
			}
			return true
		l78:
			position, tokenIndex, depth = position78, tokenIndex78, depth78
			return false
		},
		/* 21 Comparison <- <<ComparisonOp>> */
		nil,
		/* 22 ComparisonOp <- <('<' / ('>' '=') / ((&('>') '>') | (&('=') '=') | (&('<') ('<' '='))))> */
		nil,
		/* 23 Limit <- <('L' 'I' 'M' 'I' 'T' WS UInt)> */
		nil,
		/* 24 UInt <- <<[0-9]+>> */
		func() bool {
			position88, tokenIndex88, depth88 := position, tokenIndex, depth
			{
				position89 := position
				depth++
				{
					position90 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l88
					}
					position++
				l91:
					{
						position92, tokenIndex92, depth92 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l92
						}
						position++
						goto l91
					l92:
						position, tokenIndex, depth = position92, tokenIndex92, depth92
					}
					depth--
					add(rulePegText, position90)
				}
				depth--
				add(ruleUInt, position89)
			}
			return true
		l88:
			position, tokenIndex, depth = position88, tokenIndex88, depth88
			return false
		},
		/* 25 WS <- <WhiteSpace+> */
		func() bool {
			position93, tokenIndex93, depth93 := position, tokenIndex, depth
			{
				position94 := position
				depth++
				if !_rules[ruleWhiteSpace]() {
					goto l93
				}
			l95:
				{
					position96, tokenIndex96, depth96 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l96
					}
					goto l95
				l96:
					position, tokenIndex, depth = position96, tokenIndex96, depth96
				}
				depth--
				add(ruleWS, position94)
			}
			return true
		l93:
			position, tokenIndex, depth = position93, tokenIndex93, depth93
			return false
		},
		/* 26 WSX <- <WhiteSpace*> */
		func() bool {
			{
				position98 := position
				depth++
			l99:
				{
					position100, tokenIndex100, depth100 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l100
					}
					goto l99
				l100:
					position, tokenIndex, depth = position100, tokenIndex100, depth100
				}
				depth--
				add(ruleWSX, position98)
			}
			return true
		},
		/* 27 WhiteSpace <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') EOL))> */
		func() bool {
			position101, tokenIndex101, depth101 := position, tokenIndex, depth
			{
				position102 := position
				depth++
				{
					switch buffer[position] {
					case '\t':
						if buffer[position] != rune('\t') {
							goto l101
						}
						position++
						break
					case ' ':
						if buffer[position] != rune(' ') {
							goto l101
						}
						position++
						break
					default:
						{
							position104 := position
							depth++
							{
								position105, tokenIndex105, depth105 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l106
								}
								position++
								if buffer[position] != rune('\n') {
									goto l106
								}
								position++
								goto l105
							l106:
								position, tokenIndex, depth = position105, tokenIndex105, depth105
								if buffer[position] != rune('\n') {
									goto l107
								}
								position++
								goto l105
							l107:
								position, tokenIndex, depth = position105, tokenIndex105, depth105
								if buffer[position] != rune('\r') {
									goto l101
								}
								position++
							}
						l105:
							depth--
							add(ruleEOL, position104)
						}
						break
					}
				}

				depth--
				add(ruleWhiteSpace, position102)
			}
			return true
		l101:
			position, tokenIndex, depth = position101, tokenIndex101, depth101
			return false
		},
		/* 28 EOL <- <(('\r' '\n') / '\n' / '\r')> */
		nil,
		/* 29 EOF <- <!.> */
		nil,
		nil,
	}
	p.rules = _rules
}
