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
	ruleSimpleCriteria
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
	"NamespacePart",
	"Wildcard",
	"Criteria",
	"MultiCriteria",
	"SimpleCriteria",
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
	rules  [30]func() bool
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
						{
							position22 := position
							depth++
							if !_rules[ruleSimpleCriteria]() {
								goto l19
							}
						l23:
							{
								position24, tokenIndex24, depth24 := position, tokenIndex, depth
								if !_rules[ruleWS]() {
									goto l24
								}
								if buffer[position] != rune('A') {
									goto l24
								}
								position++
								if buffer[position] != rune('N') {
									goto l24
								}
								position++
								if buffer[position] != rune('D') {
									goto l24
								}
								position++
								if !_rules[ruleWS]() {
									goto l24
								}
								if !_rules[ruleSimpleCriteria]() {
									goto l24
								}
								goto l23
							l24:
								position, tokenIndex, depth = position24, tokenIndex24, depth24
							}
							depth--
							add(ruleMultiCriteria, position22)
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
					position25, tokenIndex25, depth25 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l25
					}
					{
						position27 := position
						depth++
						if buffer[position] != rune('L') {
							goto l25
						}
						position++
						if buffer[position] != rune('I') {
							goto l25
						}
						position++
						if buffer[position] != rune('M') {
							goto l25
						}
						position++
						if buffer[position] != rune('I') {
							goto l25
						}
						position++
						if buffer[position] != rune('T') {
							goto l25
						}
						position++
						if !_rules[ruleWS]() {
							goto l25
						}
						if !_rules[ruleUInt]() {
							goto l25
						}
						depth--
						add(ruleLimit, position27)
					}
					goto l26
				l25:
					position, tokenIndex, depth = position25, tokenIndex25, depth25
				}
			l26:
				if !_rules[ruleWSX]() {
					goto l0
				}
				{
					position28 := position
					depth++
					{
						position29, tokenIndex29, depth29 := position, tokenIndex, depth
						if !matchDot() {
							goto l29
						}
						goto l0
					l29:
						position, tokenIndex, depth = position29, tokenIndex29, depth29
					}
					depth--
					add(ruleEOF, position28)
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
			position31, tokenIndex31, depth31 := position, tokenIndex, depth
			{
				position32 := position
				depth++
				{
					switch buffer[position] {
					case 't':
						if buffer[position] != rune('t') {
							goto l31
						}
						position++
						if buffer[position] != rune('i') {
							goto l31
						}
						position++
						if buffer[position] != rune('m') {
							goto l31
						}
						position++
						if buffer[position] != rune('e') {
							goto l31
						}
						position++
						if buffer[position] != rune('s') {
							goto l31
						}
						position++
						if buffer[position] != rune('t') {
							goto l31
						}
						position++
						if buffer[position] != rune('a') {
							goto l31
						}
						position++
						if buffer[position] != rune('m') {
							goto l31
						}
						position++
						if buffer[position] != rune('p') {
							goto l31
						}
						position++
						break
					case 's':
						if buffer[position] != rune('s') {
							goto l31
						}
						position++
						if buffer[position] != rune('o') {
							goto l31
						}
						position++
						if buffer[position] != rune('u') {
							goto l31
						}
						position++
						if buffer[position] != rune('r') {
							goto l31
						}
						position++
						if buffer[position] != rune('c') {
							goto l31
						}
						position++
						if buffer[position] != rune('e') {
							goto l31
						}
						position++
						break
					case 'n':
						if buffer[position] != rune('n') {
							goto l31
						}
						position++
						if buffer[position] != rune('a') {
							goto l31
						}
						position++
						if buffer[position] != rune('m') {
							goto l31
						}
						position++
						if buffer[position] != rune('e') {
							goto l31
						}
						position++
						if buffer[position] != rune('s') {
							goto l31
						}
						position++
						if buffer[position] != rune('p') {
							goto l31
						}
						position++
						if buffer[position] != rune('a') {
							goto l31
						}
						position++
						if buffer[position] != rune('c') {
							goto l31
						}
						position++
						if buffer[position] != rune('e') {
							goto l31
						}
						position++
						break
					case 'p':
						if buffer[position] != rune('p') {
							goto l31
						}
						position++
						if buffer[position] != rune('u') {
							goto l31
						}
						position++
						if buffer[position] != rune('b') {
							goto l31
						}
						position++
						if buffer[position] != rune('l') {
							goto l31
						}
						position++
						if buffer[position] != rune('i') {
							goto l31
						}
						position++
						if buffer[position] != rune('s') {
							goto l31
						}
						position++
						if buffer[position] != rune('h') {
							goto l31
						}
						position++
						if buffer[position] != rune('e') {
							goto l31
						}
						position++
						if buffer[position] != rune('r') {
							goto l31
						}
						position++
						break
					case 'i':
						if buffer[position] != rune('i') {
							goto l31
						}
						position++
						if buffer[position] != rune('d') {
							goto l31
						}
						position++
						break
					case 'b':
						if buffer[position] != rune('b') {
							goto l31
						}
						position++
						if buffer[position] != rune('o') {
							goto l31
						}
						position++
						if buffer[position] != rune('d') {
							goto l31
						}
						position++
						if buffer[position] != rune('y') {
							goto l31
						}
						position++
						break
					default:
						if buffer[position] != rune('*') {
							goto l31
						}
						position++
						break
					}
				}

				depth--
				add(ruleSimpleSelector, position32)
			}
			return true
		l31:
			position, tokenIndex, depth = position31, tokenIndex31, depth31
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
			position39, tokenIndex39, depth39 := position, tokenIndex, depth
			{
				position40 := position
				depth++
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

			l41:
				{
					position42, tokenIndex42, depth42 := position, tokenIndex, depth
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

					goto l41
				l42:
					position, tokenIndex, depth = position42, tokenIndex42, depth42
				}
				depth--
				add(ruleNamespacePart, position40)
			}
			return true
		l39:
			position, tokenIndex, depth = position39, tokenIndex39, depth39
			return false
		},
		/* 9 Wildcard <- <'*'> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				if buffer[position] != rune('*') {
					goto l45
				}
				position++
				depth--
				add(ruleWildcard, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 10 Criteria <- <('W' 'H' 'E' 'R' 'E' WS MultiCriteria)> */
		nil,
		/* 11 MultiCriteria <- <(SimpleCriteria (WS ('A' 'N' 'D') WS SimpleCriteria)*)> */
		nil,
		/* 12 SimpleCriteria <- <((&('t') TimeCriteria) | (&('s') SourceCriteria) | (&('p') PublisherCriteria) | (&('i') IdCriteria))> */
		func() bool {
			position49, tokenIndex49, depth49 := position, tokenIndex, depth
			{
				position50 := position
				depth++
				{
					switch buffer[position] {
					case 't':
						{
							position52 := position
							depth++
							if buffer[position] != rune('t') {
								goto l49
							}
							position++
							if buffer[position] != rune('i') {
								goto l49
							}
							position++
							if buffer[position] != rune('m') {
								goto l49
							}
							position++
							if buffer[position] != rune('e') {
								goto l49
							}
							position++
							if buffer[position] != rune('s') {
								goto l49
							}
							position++
							if buffer[position] != rune('t') {
								goto l49
							}
							position++
							if buffer[position] != rune('a') {
								goto l49
							}
							position++
							if buffer[position] != rune('m') {
								goto l49
							}
							position++
							if buffer[position] != rune('p') {
								goto l49
							}
							position++
							if !_rules[ruleWSX]() {
								goto l49
							}
							{
								position53 := position
								depth++
								{
									position54 := position
									depth++
									{
										position55 := position
										depth++
										{
											position56, tokenIndex56, depth56 := position, tokenIndex, depth
											if buffer[position] != rune('<') {
												goto l57
											}
											position++
											if buffer[position] != rune('=') {
												goto l57
											}
											position++
											goto l56
										l57:
											position, tokenIndex, depth = position56, tokenIndex56, depth56
											if buffer[position] != rune('>') {
												goto l58
											}
											position++
											if buffer[position] != rune('=') {
												goto l58
											}
											position++
											goto l56
										l58:
											position, tokenIndex, depth = position56, tokenIndex56, depth56
											{
												switch buffer[position] {
												case '>':
													if buffer[position] != rune('>') {
														goto l49
													}
													position++
													break
												case '=':
													if buffer[position] != rune('=') {
														goto l49
													}
													position++
													break
												default:
													if buffer[position] != rune('<') {
														goto l49
													}
													position++
													break
												}
											}

										}
									l56:
										depth--
										add(ruleComparisonOp, position55)
									}
									depth--
									add(rulePegText, position54)
								}
								depth--
								add(ruleComparison, position53)
							}
							if !_rules[ruleWSX]() {
								goto l49
							}
							if !_rules[ruleUInt]() {
								goto l49
							}
							depth--
							add(ruleTimeCriteria, position52)
						}
						break
					case 's':
						{
							position60 := position
							depth++
							if buffer[position] != rune('s') {
								goto l49
							}
							position++
							if buffer[position] != rune('o') {
								goto l49
							}
							position++
							if buffer[position] != rune('u') {
								goto l49
							}
							position++
							if buffer[position] != rune('r') {
								goto l49
							}
							position++
							if buffer[position] != rune('c') {
								goto l49
							}
							position++
							if buffer[position] != rune('e') {
								goto l49
							}
							position++
							if !_rules[ruleWSX]() {
								goto l49
							}
							if buffer[position] != rune('=') {
								goto l49
							}
							position++
							if !_rules[ruleWSX]() {
								goto l49
							}
							if !_rules[rulePublisherId]() {
								goto l49
							}
							depth--
							add(ruleSourceCriteria, position60)
						}
						break
					case 'p':
						{
							position61 := position
							depth++
							if buffer[position] != rune('p') {
								goto l49
							}
							position++
							if buffer[position] != rune('u') {
								goto l49
							}
							position++
							if buffer[position] != rune('b') {
								goto l49
							}
							position++
							if buffer[position] != rune('l') {
								goto l49
							}
							position++
							if buffer[position] != rune('i') {
								goto l49
							}
							position++
							if buffer[position] != rune('s') {
								goto l49
							}
							position++
							if buffer[position] != rune('h') {
								goto l49
							}
							position++
							if buffer[position] != rune('e') {
								goto l49
							}
							position++
							if buffer[position] != rune('r') {
								goto l49
							}
							position++
							if !_rules[ruleWSX]() {
								goto l49
							}
							if buffer[position] != rune('=') {
								goto l49
							}
							position++
							if !_rules[ruleWSX]() {
								goto l49
							}
							if !_rules[rulePublisherId]() {
								goto l49
							}
							depth--
							add(rulePublisherCriteria, position61)
						}
						break
					default:
						{
							position62 := position
							depth++
							if buffer[position] != rune('i') {
								goto l49
							}
							position++
							if buffer[position] != rune('d') {
								goto l49
							}
							position++
							if !_rules[ruleWSX]() {
								goto l49
							}
							if buffer[position] != rune('=') {
								goto l49
							}
							position++
							if !_rules[ruleWSX]() {
								goto l49
							}
							{
								position63 := position
								depth++
								{
									position64 := position
									depth++
									{
										switch buffer[position] {
										case ':':
											if buffer[position] != rune(':') {
												goto l49
											}
											position++
											break
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l49
											}
											position++
											break
										case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l49
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('a') || c > rune('z') {
												goto l49
											}
											position++
											break
										}
									}

								l65:
									{
										position66, tokenIndex66, depth66 := position, tokenIndex, depth
										{
											switch buffer[position] {
											case ':':
												if buffer[position] != rune(':') {
													goto l66
												}
												position++
												break
											case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
												if c := buffer[position]; c < rune('0') || c > rune('9') {
													goto l66
												}
												position++
												break
											case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
												if c := buffer[position]; c < rune('A') || c > rune('Z') {
													goto l66
												}
												position++
												break
											default:
												if c := buffer[position]; c < rune('a') || c > rune('z') {
													goto l66
												}
												position++
												break
											}
										}

										goto l65
									l66:
										position, tokenIndex, depth = position66, tokenIndex66, depth66
									}
									depth--
									add(rulePegText, position64)
								}
								depth--
								add(ruleStatementId, position63)
							}
							depth--
							add(ruleIdCriteria, position62)
						}
						break
					}
				}

				depth--
				add(ruleSimpleCriteria, position50)
			}
			return true
		l49:
			position, tokenIndex, depth = position49, tokenIndex49, depth49
			return false
		},
		/* 13 IdCriteria <- <('i' 'd' WSX '=' WSX StatementId)> */
		nil,
		/* 14 PublisherCriteria <- <('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r' WSX '=' WSX PublisherId)> */
		nil,
		/* 15 SourceCriteria <- <('s' 'o' 'u' 'r' 'c' 'e' WSX '=' WSX PublisherId)> */
		nil,
		/* 16 TimeCriteria <- <('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p' WSX Comparison WSX UInt)> */
		nil,
		/* 17 StatementId <- <<((&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 18 PublisherId <- <<((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		func() bool {
			position74, tokenIndex74, depth74 := position, tokenIndex, depth
			{
				position75 := position
				depth++
				{
					position76 := position
					depth++
					{
						switch buffer[position] {
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

				l77:
					{
						position78, tokenIndex78, depth78 := position, tokenIndex, depth
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

						goto l77
					l78:
						position, tokenIndex, depth = position78, tokenIndex78, depth78
					}
					depth--
					add(rulePegText, position76)
				}
				depth--
				add(rulePublisherId, position75)
			}
			return true
		l74:
			position, tokenIndex, depth = position74, tokenIndex74, depth74
			return false
		},
		/* 19 Comparison <- <<ComparisonOp>> */
		nil,
		/* 20 ComparisonOp <- <(('<' '=') / ('>' '=') / ((&('>') '>') | (&('=') '=') | (&('<') '<')))> */
		nil,
		/* 21 Limit <- <('L' 'I' 'M' 'I' 'T' WS UInt)> */
		nil,
		/* 22 UInt <- <<[0-9]+>> */
		func() bool {
			position84, tokenIndex84, depth84 := position, tokenIndex, depth
			{
				position85 := position
				depth++
				{
					position86 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l84
					}
					position++
				l87:
					{
						position88, tokenIndex88, depth88 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l88
						}
						position++
						goto l87
					l88:
						position, tokenIndex, depth = position88, tokenIndex88, depth88
					}
					depth--
					add(rulePegText, position86)
				}
				depth--
				add(ruleUInt, position85)
			}
			return true
		l84:
			position, tokenIndex, depth = position84, tokenIndex84, depth84
			return false
		},
		/* 23 WS <- <WhiteSpace+> */
		func() bool {
			position89, tokenIndex89, depth89 := position, tokenIndex, depth
			{
				position90 := position
				depth++
				if !_rules[ruleWhiteSpace]() {
					goto l89
				}
			l91:
				{
					position92, tokenIndex92, depth92 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l92
					}
					goto l91
				l92:
					position, tokenIndex, depth = position92, tokenIndex92, depth92
				}
				depth--
				add(ruleWS, position90)
			}
			return true
		l89:
			position, tokenIndex, depth = position89, tokenIndex89, depth89
			return false
		},
		/* 24 WSX <- <WhiteSpace*> */
		func() bool {
			{
				position94 := position
				depth++
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
				add(ruleWSX, position94)
			}
			return true
		},
		/* 25 WhiteSpace <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') EOL))> */
		func() bool {
			position97, tokenIndex97, depth97 := position, tokenIndex, depth
			{
				position98 := position
				depth++
				{
					switch buffer[position] {
					case '\t':
						if buffer[position] != rune('\t') {
							goto l97
						}
						position++
						break
					case ' ':
						if buffer[position] != rune(' ') {
							goto l97
						}
						position++
						break
					default:
						{
							position100 := position
							depth++
							{
								position101, tokenIndex101, depth101 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l102
								}
								position++
								if buffer[position] != rune('\n') {
									goto l102
								}
								position++
								goto l101
							l102:
								position, tokenIndex, depth = position101, tokenIndex101, depth101
								if buffer[position] != rune('\n') {
									goto l103
								}
								position++
								goto l101
							l103:
								position, tokenIndex, depth = position101, tokenIndex101, depth101
								if buffer[position] != rune('\r') {
									goto l97
								}
								position++
							}
						l101:
							depth--
							add(ruleEOL, position100)
						}
						break
					}
				}

				depth--
				add(ruleWhiteSpace, position98)
			}
			return true
		l97:
			position, tokenIndex, depth = position97, tokenIndex97, depth97
			return false
		},
		/* 26 EOL <- <(('\r' '\n') / '\n' / '\r')> */
		nil,
		/* 27 EOF <- <!.> */
		nil,
		nil,
	}
	p.rules = _rules
}
