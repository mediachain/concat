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
	ruleValueCriteria
	ruleIdCriteria
	rulePublisherCriteria
	ruleSourceCriteria
	ruleTimeCriteria
	ruleBoolean
	ruleBooleanOp
	ruleComparison
	ruleComparisonOp
	ruleLimit
	ruleStatementId
	rulePublisherId
	ruleUInt
	ruleWS
	ruleWSX
	ruleWhiteSpace
	ruleEOL
	ruleEOF
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12

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
	"ValueCriteria",
	"IdCriteria",
	"PublisherCriteria",
	"SourceCriteria",
	"TimeCriteria",
	"Boolean",
	"BooleanOp",
	"Comparison",
	"ComparisonOp",
	"Limit",
	"StatementId",
	"PublisherId",
	"UInt",
	"WS",
	"WSX",
	"WhiteSpace",
	"EOL",
	"EOF",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",

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
	*ParseState

	Buffer string
	buffer []rune
	rules  [49]func() bool
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

func (p *QueryParser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.setCriteria()
		case ruleAction1:
			p.addCriteriaCombinator()
		case ruleAction2:
			p.addValueCriteria()
		case ruleAction3:
			p.addTimeCriteria()
		case ruleAction4:
			p.push(text)
		case ruleAction5:
			p.push(text)
		case ruleAction6:
			p.push(text)
		case ruleAction7:
			p.push(text)
		case ruleAction8:
			p.push(text)
		case ruleAction9:
			p.push(text)
		case ruleAction10:
			p.push(text)
		case ruleAction11:
			p.push(text)
		case ruleAction12:
			p.push(text)

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
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
						{
							add(ruleAction0, position)
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
		/* 2 SimpleSelector <- <<SimpleSelectorOp>> */
		func() bool {
			position31, tokenIndex31, depth31 := position, tokenIndex, depth
			{
				position32 := position
				depth++
				{
					position33 := position
					depth++
					{
						position34 := position
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
						add(ruleSimpleSelectorOp, position34)
					}
					depth--
					add(rulePegText, position33)
				}
				depth--
				add(ruleSimpleSelector, position32)
			}
			return true
		l31:
			position, tokenIndex, depth = position31, tokenIndex31, depth31
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
			position43, tokenIndex43, depth43 := position, tokenIndex, depth
			{
				position44 := position
				depth++
				{
					switch buffer[position] {
					case '-':
						if buffer[position] != rune('-') {
							goto l43
						}
						position++
						break
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
						case '-':
							if buffer[position] != rune('-') {
								goto l46
							}
							position++
							break
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
		/* 11 Wildcard <- <'*'> */
		func() bool {
			position49, tokenIndex49, depth49 := position, tokenIndex, depth
			{
				position50 := position
				depth++
				if buffer[position] != rune('*') {
					goto l49
				}
				position++
				depth--
				add(ruleWildcard, position50)
			}
			return true
		l49:
			position, tokenIndex, depth = position49, tokenIndex49, depth49
			return false
		},
		/* 12 Criteria <- <('W' 'H' 'E' 'R' 'E' WS MultiCriteria Action0)> */
		nil,
		/* 13 MultiCriteria <- <(CompoundCriteria (WS Boolean WS CompoundCriteria Action1)*)> */
		func() bool {
			position52, tokenIndex52, depth52 := position, tokenIndex, depth
			{
				position53 := position
				depth++
				if !_rules[ruleCompoundCriteria]() {
					goto l52
				}
			l54:
				{
					position55, tokenIndex55, depth55 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l55
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
									if buffer[position] != rune('A') {
										goto l60
									}
									position++
									if buffer[position] != rune('N') {
										goto l60
									}
									position++
									if buffer[position] != rune('D') {
										goto l60
									}
									position++
									goto l59
								l60:
									position, tokenIndex, depth = position59, tokenIndex59, depth59
									if buffer[position] != rune('O') {
										goto l55
									}
									position++
									if buffer[position] != rune('R') {
										goto l55
									}
									position++
								}
							l59:
								depth--
								add(ruleBooleanOp, position58)
							}
							depth--
							add(rulePegText, position57)
						}
						{
							add(ruleAction11, position)
						}
						depth--
						add(ruleBoolean, position56)
					}
					if !_rules[ruleWS]() {
						goto l55
					}
					if !_rules[ruleCompoundCriteria]() {
						goto l55
					}
					{
						add(ruleAction1, position)
					}
					goto l54
				l55:
					position, tokenIndex, depth = position55, tokenIndex55, depth55
				}
				depth--
				add(ruleMultiCriteria, position53)
			}
			return true
		l52:
			position, tokenIndex, depth = position52, tokenIndex52, depth52
			return false
		},
		/* 14 CompoundCriteria <- <(SimpleCriteria / ('(' MultiCriteria ')'))> */
		func() bool {
			position63, tokenIndex63, depth63 := position, tokenIndex, depth
			{
				position64 := position
				depth++
				{
					position65, tokenIndex65, depth65 := position, tokenIndex, depth
					{
						position67 := position
						depth++
						{
							position68, tokenIndex68, depth68 := position, tokenIndex, depth
							{
								position70 := position
								depth++
								{
									switch buffer[position] {
									case 's':
										{
											position72 := position
											depth++
											{
												position73 := position
												depth++
												if buffer[position] != rune('s') {
													goto l69
												}
												position++
												if buffer[position] != rune('o') {
													goto l69
												}
												position++
												if buffer[position] != rune('u') {
													goto l69
												}
												position++
												if buffer[position] != rune('r') {
													goto l69
												}
												position++
												if buffer[position] != rune('c') {
													goto l69
												}
												position++
												if buffer[position] != rune('e') {
													goto l69
												}
												position++
												depth--
												add(rulePegText, position73)
											}
											{
												add(ruleAction8, position)
											}
											if !_rules[ruleWSX]() {
												goto l69
											}
											if buffer[position] != rune('=') {
												goto l69
											}
											position++
											if !_rules[ruleWSX]() {
												goto l69
											}
											if !_rules[rulePublisherId]() {
												goto l69
											}
											{
												add(ruleAction9, position)
											}
											depth--
											add(ruleSourceCriteria, position72)
										}
										break
									case 'p':
										{
											position76 := position
											depth++
											{
												position77 := position
												depth++
												if buffer[position] != rune('p') {
													goto l69
												}
												position++
												if buffer[position] != rune('u') {
													goto l69
												}
												position++
												if buffer[position] != rune('b') {
													goto l69
												}
												position++
												if buffer[position] != rune('l') {
													goto l69
												}
												position++
												if buffer[position] != rune('i') {
													goto l69
												}
												position++
												if buffer[position] != rune('s') {
													goto l69
												}
												position++
												if buffer[position] != rune('h') {
													goto l69
												}
												position++
												if buffer[position] != rune('e') {
													goto l69
												}
												position++
												if buffer[position] != rune('r') {
													goto l69
												}
												position++
												depth--
												add(rulePegText, position77)
											}
											{
												add(ruleAction6, position)
											}
											if !_rules[ruleWSX]() {
												goto l69
											}
											if buffer[position] != rune('=') {
												goto l69
											}
											position++
											if !_rules[ruleWSX]() {
												goto l69
											}
											if !_rules[rulePublisherId]() {
												goto l69
											}
											{
												add(ruleAction7, position)
											}
											depth--
											add(rulePublisherCriteria, position76)
										}
										break
									default:
										{
											position80 := position
											depth++
											{
												position81 := position
												depth++
												if buffer[position] != rune('i') {
													goto l69
												}
												position++
												if buffer[position] != rune('d') {
													goto l69
												}
												position++
												depth--
												add(rulePegText, position81)
											}
											{
												add(ruleAction4, position)
											}
											if !_rules[ruleWSX]() {
												goto l69
											}
											if buffer[position] != rune('=') {
												goto l69
											}
											position++
											if !_rules[ruleWSX]() {
												goto l69
											}
											{
												position83 := position
												depth++
												{
													position84 := position
													depth++
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

												l85:
													{
														position86, tokenIndex86, depth86 := position, tokenIndex, depth
														{
															switch buffer[position] {
															case ':':
																if buffer[position] != rune(':') {
																	goto l86
																}
																position++
																break
															case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
																if c := buffer[position]; c < rune('0') || c > rune('9') {
																	goto l86
																}
																position++
																break
															case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
																if c := buffer[position]; c < rune('A') || c > rune('Z') {
																	goto l86
																}
																position++
																break
															default:
																if c := buffer[position]; c < rune('a') || c > rune('z') {
																	goto l86
																}
																position++
																break
															}
														}

														goto l85
													l86:
														position, tokenIndex, depth = position86, tokenIndex86, depth86
													}
													depth--
													add(rulePegText, position84)
												}
												depth--
												add(ruleStatementId, position83)
											}
											{
												add(ruleAction5, position)
											}
											depth--
											add(ruleIdCriteria, position80)
										}
										break
									}
								}

								depth--
								add(ruleValueCriteria, position70)
							}
							{
								add(ruleAction2, position)
							}
							goto l68
						l69:
							position, tokenIndex, depth = position68, tokenIndex68, depth68
							{
								position91 := position
								depth++
								if buffer[position] != rune('t') {
									goto l66
								}
								position++
								if buffer[position] != rune('i') {
									goto l66
								}
								position++
								if buffer[position] != rune('m') {
									goto l66
								}
								position++
								if buffer[position] != rune('e') {
									goto l66
								}
								position++
								if buffer[position] != rune('s') {
									goto l66
								}
								position++
								if buffer[position] != rune('t') {
									goto l66
								}
								position++
								if buffer[position] != rune('a') {
									goto l66
								}
								position++
								if buffer[position] != rune('m') {
									goto l66
								}
								position++
								if buffer[position] != rune('p') {
									goto l66
								}
								position++
								if !_rules[ruleWSX]() {
									goto l66
								}
								{
									position92 := position
									depth++
									{
										position93 := position
										depth++
										{
											position94 := position
											depth++
											{
												position95, tokenIndex95, depth95 := position, tokenIndex, depth
												if buffer[position] != rune('<') {
													goto l96
												}
												position++
												if buffer[position] != rune('=') {
													goto l96
												}
												position++
												goto l95
											l96:
												position, tokenIndex, depth = position95, tokenIndex95, depth95
												if buffer[position] != rune('>') {
													goto l97
												}
												position++
												if buffer[position] != rune('=') {
													goto l97
												}
												position++
												goto l95
											l97:
												position, tokenIndex, depth = position95, tokenIndex95, depth95
												{
													switch buffer[position] {
													case '>':
														if buffer[position] != rune('>') {
															goto l66
														}
														position++
														break
													case '=':
														if buffer[position] != rune('=') {
															goto l66
														}
														position++
														break
													default:
														if buffer[position] != rune('<') {
															goto l66
														}
														position++
														break
													}
												}

											}
										l95:
											depth--
											add(ruleComparisonOp, position94)
										}
										depth--
										add(rulePegText, position93)
									}
									{
										add(ruleAction12, position)
									}
									depth--
									add(ruleComparison, position92)
								}
								if !_rules[ruleWSX]() {
									goto l66
								}
								if !_rules[ruleUInt]() {
									goto l66
								}
								{
									add(ruleAction10, position)
								}
								depth--
								add(ruleTimeCriteria, position91)
							}
							{
								add(ruleAction3, position)
							}
						}
					l68:
						depth--
						add(ruleSimpleCriteria, position67)
					}
					goto l65
				l66:
					position, tokenIndex, depth = position65, tokenIndex65, depth65
					if buffer[position] != rune('(') {
						goto l63
					}
					position++
					if !_rules[ruleMultiCriteria]() {
						goto l63
					}
					if buffer[position] != rune(')') {
						goto l63
					}
					position++
				}
			l65:
				depth--
				add(ruleCompoundCriteria, position64)
			}
			return true
		l63:
			position, tokenIndex, depth = position63, tokenIndex63, depth63
			return false
		},
		/* 15 SimpleCriteria <- <((ValueCriteria Action2) / (TimeCriteria Action3))> */
		nil,
		/* 16 ValueCriteria <- <((&('s') SourceCriteria) | (&('p') PublisherCriteria) | (&('i') IdCriteria))> */
		nil,
		/* 17 IdCriteria <- <(<('i' 'd')> Action4 WSX '=' WSX StatementId Action5)> */
		nil,
		/* 18 PublisherCriteria <- <(<('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')> Action6 WSX '=' WSX PublisherId Action7)> */
		nil,
		/* 19 SourceCriteria <- <(<('s' 'o' 'u' 'r' 'c' 'e')> Action8 WSX '=' WSX PublisherId Action9)> */
		nil,
		/* 20 TimeCriteria <- <('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p' WSX Comparison WSX UInt Action10)> */
		nil,
		/* 21 Boolean <- <(<BooleanOp> Action11)> */
		nil,
		/* 22 BooleanOp <- <(('A' 'N' 'D') / ('O' 'R'))> */
		nil,
		/* 23 Comparison <- <(<ComparisonOp> Action12)> */
		nil,
		/* 24 ComparisonOp <- <(('<' '=') / ('>' '=') / ((&('>') '>') | (&('=') '=') | (&('<') '<')))> */
		nil,
		/* 25 Limit <- <('L' 'I' 'M' 'I' 'T' WS UInt)> */
		nil,
		/* 26 StatementId <- <<((&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 27 PublisherId <- <<((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		func() bool {
			position114, tokenIndex114, depth114 := position, tokenIndex, depth
			{
				position115 := position
				depth++
				{
					position116 := position
					depth++
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l114
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l114
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l114
							}
							position++
							break
						}
					}

				l117:
					{
						position118, tokenIndex118, depth118 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l118
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l118
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l118
								}
								position++
								break
							}
						}

						goto l117
					l118:
						position, tokenIndex, depth = position118, tokenIndex118, depth118
					}
					depth--
					add(rulePegText, position116)
				}
				depth--
				add(rulePublisherId, position115)
			}
			return true
		l114:
			position, tokenIndex, depth = position114, tokenIndex114, depth114
			return false
		},
		/* 28 UInt <- <<[0-9]+>> */
		func() bool {
			position121, tokenIndex121, depth121 := position, tokenIndex, depth
			{
				position122 := position
				depth++
				{
					position123 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l121
					}
					position++
				l124:
					{
						position125, tokenIndex125, depth125 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l125
						}
						position++
						goto l124
					l125:
						position, tokenIndex, depth = position125, tokenIndex125, depth125
					}
					depth--
					add(rulePegText, position123)
				}
				depth--
				add(ruleUInt, position122)
			}
			return true
		l121:
			position, tokenIndex, depth = position121, tokenIndex121, depth121
			return false
		},
		/* 29 WS <- <WhiteSpace+> */
		func() bool {
			position126, tokenIndex126, depth126 := position, tokenIndex, depth
			{
				position127 := position
				depth++
				if !_rules[ruleWhiteSpace]() {
					goto l126
				}
			l128:
				{
					position129, tokenIndex129, depth129 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l129
					}
					goto l128
				l129:
					position, tokenIndex, depth = position129, tokenIndex129, depth129
				}
				depth--
				add(ruleWS, position127)
			}
			return true
		l126:
			position, tokenIndex, depth = position126, tokenIndex126, depth126
			return false
		},
		/* 30 WSX <- <WhiteSpace*> */
		func() bool {
			{
				position131 := position
				depth++
			l132:
				{
					position133, tokenIndex133, depth133 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l133
					}
					goto l132
				l133:
					position, tokenIndex, depth = position133, tokenIndex133, depth133
				}
				depth--
				add(ruleWSX, position131)
			}
			return true
		},
		/* 31 WhiteSpace <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') EOL))> */
		func() bool {
			position134, tokenIndex134, depth134 := position, tokenIndex, depth
			{
				position135 := position
				depth++
				{
					switch buffer[position] {
					case '\t':
						if buffer[position] != rune('\t') {
							goto l134
						}
						position++
						break
					case ' ':
						if buffer[position] != rune(' ') {
							goto l134
						}
						position++
						break
					default:
						{
							position137 := position
							depth++
							{
								position138, tokenIndex138, depth138 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l139
								}
								position++
								if buffer[position] != rune('\n') {
									goto l139
								}
								position++
								goto l138
							l139:
								position, tokenIndex, depth = position138, tokenIndex138, depth138
								if buffer[position] != rune('\n') {
									goto l140
								}
								position++
								goto l138
							l140:
								position, tokenIndex, depth = position138, tokenIndex138, depth138
								if buffer[position] != rune('\r') {
									goto l134
								}
								position++
							}
						l138:
							depth--
							add(ruleEOL, position137)
						}
						break
					}
				}

				depth--
				add(ruleWhiteSpace, position135)
			}
			return true
		l134:
			position, tokenIndex, depth = position134, tokenIndex134, depth134
			return false
		},
		/* 32 EOL <- <(('\r' '\n') / '\n' / '\r')> */
		nil,
		/* 33 EOF <- <!.> */
		nil,
		nil,
		/* 36 Action0 <- <{ p.setCriteria() }> */
		nil,
		/* 37 Action1 <- <{ p.addCriteriaCombinator() }> */
		nil,
		/* 38 Action2 <- <{ p.addValueCriteria() }> */
		nil,
		/* 39 Action3 <- <{ p.addTimeCriteria() }> */
		nil,
		/* 40 Action4 <- <{ p.push(text) }> */
		nil,
		/* 41 Action5 <- <{ p.push(text) }> */
		nil,
		/* 42 Action6 <- <{ p.push(text) }> */
		nil,
		/* 43 Action7 <- <{ p.push(text) }> */
		nil,
		/* 44 Action8 <- <{ p.push(text) }> */
		nil,
		/* 45 Action9 <- <{ p.push(text) }> */
		nil,
		/* 46 Action10 <- <{ p.push(text) }> */
		nil,
		/* 47 Action11 <- <{ p.push(text) }> */
		nil,
		/* 48 Action12 <- <{ p.push(text) }> */
		nil,
	}
	p.rules = _rules
}
