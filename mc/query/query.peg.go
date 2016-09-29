package query

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
	ruleSelect
	ruleDelete
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
	ruleValueCompare
	ruleValueCompareOp
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
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	rulePegText
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19
	ruleAction20
	ruleAction21
	ruleAction22
	ruleAction23

	rulePre
	ruleIn
	ruleSuf
)

var rul3s = [...]string{
	"Unknown",
	"Grammar",
	"Select",
	"Delete",
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
	"ValueCompare",
	"ValueCompareOp",
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
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"PegText",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",
	"Action20",
	"Action21",
	"Action22",
	"Action23",

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
	rules  [64]func() bool
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
			p.setSelectOp()
		case ruleAction1:
			p.setDeleteOp()
		case ruleAction2:
			p.setSimpleSelector()
		case ruleAction3:
			p.setCompoundSelector()
		case ruleAction4:
			p.setFunctionSelector()
		case ruleAction5:
			p.push(text)
		case ruleAction6:
			p.push(text)
		case ruleAction7:
			p.setNamespace(text)
		case ruleAction8:
			p.setCriteria()
		case ruleAction9:
			p.addCompoundCriteria()
		case ruleAction10:
			p.addNegatedCriteria()
		case ruleAction11:
			p.addValueCriteria()
		case ruleAction12:
			p.addTimeCriteria()
		case ruleAction13:
			p.push(text)
		case ruleAction14:
			p.push(text)
		case ruleAction15:
			p.push(text)
		case ruleAction16:
			p.push(text)
		case ruleAction17:
			p.push(text)
		case ruleAction18:
			p.push(text)
		case ruleAction19:
			p.push(text)
		case ruleAction20:
			p.push(text)
		case ruleAction21:
			p.push(text)
		case ruleAction22:
			p.push(text)
		case ruleAction23:
			p.setLimit(text)

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
		/* 0 Grammar <- <((Select WSX EOF Action0) / (Delete WSX EOF Action1))> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				{
					position2, tokenIndex2, depth2 := position, tokenIndex, depth
					{
						position4 := position
						depth++
						if buffer[position] != rune('S') {
							goto l3
						}
						position++
						if buffer[position] != rune('E') {
							goto l3
						}
						position++
						if buffer[position] != rune('L') {
							goto l3
						}
						position++
						if buffer[position] != rune('E') {
							goto l3
						}
						position++
						if buffer[position] != rune('C') {
							goto l3
						}
						position++
						if buffer[position] != rune('T') {
							goto l3
						}
						position++
						if !_rules[ruleWS]() {
							goto l3
						}
						{
							position5 := position
							depth++
							{
								switch buffer[position] {
								case 'C', 'M':
									{
										position7 := position
										depth++
										{
											position8 := position
											depth++
											{
												position9 := position
												depth++
												{
													position10 := position
													depth++
													{
														position11, tokenIndex11, depth11 := position, tokenIndex, depth
														if buffer[position] != rune('C') {
															goto l12
														}
														position++
														if buffer[position] != rune('O') {
															goto l12
														}
														position++
														if buffer[position] != rune('U') {
															goto l12
														}
														position++
														if buffer[position] != rune('N') {
															goto l12
														}
														position++
														if buffer[position] != rune('T') {
															goto l12
														}
														position++
														goto l11
													l12:
														position, tokenIndex, depth = position11, tokenIndex11, depth11
														if buffer[position] != rune('M') {
															goto l13
														}
														position++
														if buffer[position] != rune('I') {
															goto l13
														}
														position++
														if buffer[position] != rune('N') {
															goto l13
														}
														position++
														goto l11
													l13:
														position, tokenIndex, depth = position11, tokenIndex11, depth11
														if buffer[position] != rune('M') {
															goto l3
														}
														position++
														if buffer[position] != rune('A') {
															goto l3
														}
														position++
														if buffer[position] != rune('X') {
															goto l3
														}
														position++
													}
												l11:
													depth--
													add(ruleFunctionOp, position10)
												}
												depth--
												add(rulePegText, position9)
											}
											{
												add(ruleAction6, position)
											}
											depth--
											add(ruleFunction, position8)
										}
										if buffer[position] != rune('(') {
											goto l3
										}
										position++
										if !_rules[ruleSimpleSelector]() {
											goto l3
										}
										if buffer[position] != rune(')') {
											goto l3
										}
										position++
										depth--
										add(ruleFunctionSelector, position7)
									}
									{
										add(ruleAction4, position)
									}
									break
								case '(':
									{
										position16 := position
										depth++
										if buffer[position] != rune('(') {
											goto l3
										}
										position++
										if !_rules[ruleSimpleSelector]() {
											goto l3
										}
									l17:
										{
											position18, tokenIndex18, depth18 := position, tokenIndex, depth
											if buffer[position] != rune(',') {
												goto l18
											}
											position++
											if !_rules[ruleWSX]() {
												goto l18
											}
											if !_rules[ruleSimpleSelector]() {
												goto l18
											}
											goto l17
										l18:
											position, tokenIndex, depth = position18, tokenIndex18, depth18
										}
										if buffer[position] != rune(')') {
											goto l3
										}
										position++
										depth--
										add(ruleCompoundSelector, position16)
									}
									{
										add(ruleAction3, position)
									}
									break
								default:
									if !_rules[ruleSimpleSelector]() {
										goto l3
									}
									{
										add(ruleAction2, position)
									}
									break
								}
							}

							depth--
							add(ruleSelector, position5)
						}
						if !_rules[ruleWS]() {
							goto l3
						}
						if !_rules[ruleSource]() {
							goto l3
						}
						{
							position21, tokenIndex21, depth21 := position, tokenIndex, depth
							if !_rules[ruleWS]() {
								goto l21
							}
							if !_rules[ruleCriteria]() {
								goto l21
							}
							goto l22
						l21:
							position, tokenIndex, depth = position21, tokenIndex21, depth21
						}
					l22:
						{
							position23, tokenIndex23, depth23 := position, tokenIndex, depth
							if !_rules[ruleWS]() {
								goto l23
							}
							{
								position25 := position
								depth++
								if buffer[position] != rune('L') {
									goto l23
								}
								position++
								if buffer[position] != rune('I') {
									goto l23
								}
								position++
								if buffer[position] != rune('M') {
									goto l23
								}
								position++
								if buffer[position] != rune('I') {
									goto l23
								}
								position++
								if buffer[position] != rune('T') {
									goto l23
								}
								position++
								if !_rules[ruleWS]() {
									goto l23
								}
								if !_rules[ruleUInt]() {
									goto l23
								}
								{
									add(ruleAction23, position)
								}
								depth--
								add(ruleLimit, position25)
							}
							goto l24
						l23:
							position, tokenIndex, depth = position23, tokenIndex23, depth23
						}
					l24:
						depth--
						add(ruleSelect, position4)
					}
					if !_rules[ruleWSX]() {
						goto l3
					}
					if !_rules[ruleEOF]() {
						goto l3
					}
					{
						add(ruleAction0, position)
					}
					goto l2
				l3:
					position, tokenIndex, depth = position2, tokenIndex2, depth2
					{
						position28 := position
						depth++
						if buffer[position] != rune('D') {
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
						if buffer[position] != rune('T') {
							goto l0
						}
						position++
						if buffer[position] != rune('E') {
							goto l0
						}
						position++
						if !_rules[ruleWS]() {
							goto l0
						}
						if !_rules[ruleSource]() {
							goto l0
						}
						{
							position29, tokenIndex29, depth29 := position, tokenIndex, depth
							if !_rules[ruleWS]() {
								goto l29
							}
							if !_rules[ruleCriteria]() {
								goto l29
							}
							goto l30
						l29:
							position, tokenIndex, depth = position29, tokenIndex29, depth29
						}
					l30:
						depth--
						add(ruleDelete, position28)
					}
					if !_rules[ruleWSX]() {
						goto l0
					}
					if !_rules[ruleEOF]() {
						goto l0
					}
					{
						add(ruleAction1, position)
					}
				}
			l2:
				depth--
				add(ruleGrammar, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 Select <- <('S' 'E' 'L' 'E' 'C' 'T' WS Selector WS Source (WS Criteria)? (WS Limit)?)> */
		nil,
		/* 2 Delete <- <('D' 'E' 'L' 'E' 'T' 'E' WS Source (WS Criteria)?)> */
		nil,
		/* 3 Selector <- <((&('C' | 'M') (FunctionSelector Action4)) | (&('(') (CompoundSelector Action3)) | (&('*' | 'b' | 'i' | 'n' | 'p' | 's' | 't') (SimpleSelector Action2)))> */
		nil,
		/* 4 SimpleSelector <- <(<SimpleSelectorOp> Action5)> */
		func() bool {
			position35, tokenIndex35, depth35 := position, tokenIndex, depth
			{
				position36 := position
				depth++
				{
					position37 := position
					depth++
					{
						position38 := position
						depth++
						{
							switch buffer[position] {
							case 't':
								if buffer[position] != rune('t') {
									goto l35
								}
								position++
								if buffer[position] != rune('i') {
									goto l35
								}
								position++
								if buffer[position] != rune('m') {
									goto l35
								}
								position++
								if buffer[position] != rune('e') {
									goto l35
								}
								position++
								if buffer[position] != rune('s') {
									goto l35
								}
								position++
								if buffer[position] != rune('t') {
									goto l35
								}
								position++
								if buffer[position] != rune('a') {
									goto l35
								}
								position++
								if buffer[position] != rune('m') {
									goto l35
								}
								position++
								if buffer[position] != rune('p') {
									goto l35
								}
								position++
								break
							case 's':
								if buffer[position] != rune('s') {
									goto l35
								}
								position++
								if buffer[position] != rune('o') {
									goto l35
								}
								position++
								if buffer[position] != rune('u') {
									goto l35
								}
								position++
								if buffer[position] != rune('r') {
									goto l35
								}
								position++
								if buffer[position] != rune('c') {
									goto l35
								}
								position++
								if buffer[position] != rune('e') {
									goto l35
								}
								position++
								break
							case 'n':
								if buffer[position] != rune('n') {
									goto l35
								}
								position++
								if buffer[position] != rune('a') {
									goto l35
								}
								position++
								if buffer[position] != rune('m') {
									goto l35
								}
								position++
								if buffer[position] != rune('e') {
									goto l35
								}
								position++
								if buffer[position] != rune('s') {
									goto l35
								}
								position++
								if buffer[position] != rune('p') {
									goto l35
								}
								position++
								if buffer[position] != rune('a') {
									goto l35
								}
								position++
								if buffer[position] != rune('c') {
									goto l35
								}
								position++
								if buffer[position] != rune('e') {
									goto l35
								}
								position++
								break
							case 'p':
								if buffer[position] != rune('p') {
									goto l35
								}
								position++
								if buffer[position] != rune('u') {
									goto l35
								}
								position++
								if buffer[position] != rune('b') {
									goto l35
								}
								position++
								if buffer[position] != rune('l') {
									goto l35
								}
								position++
								if buffer[position] != rune('i') {
									goto l35
								}
								position++
								if buffer[position] != rune('s') {
									goto l35
								}
								position++
								if buffer[position] != rune('h') {
									goto l35
								}
								position++
								if buffer[position] != rune('e') {
									goto l35
								}
								position++
								if buffer[position] != rune('r') {
									goto l35
								}
								position++
								break
							case 'i':
								if buffer[position] != rune('i') {
									goto l35
								}
								position++
								if buffer[position] != rune('d') {
									goto l35
								}
								position++
								break
							case 'b':
								if buffer[position] != rune('b') {
									goto l35
								}
								position++
								if buffer[position] != rune('o') {
									goto l35
								}
								position++
								if buffer[position] != rune('d') {
									goto l35
								}
								position++
								if buffer[position] != rune('y') {
									goto l35
								}
								position++
								break
							default:
								if buffer[position] != rune('*') {
									goto l35
								}
								position++
								break
							}
						}

						depth--
						add(ruleSimpleSelectorOp, position38)
					}
					depth--
					add(rulePegText, position37)
				}
				{
					add(ruleAction5, position)
				}
				depth--
				add(ruleSimpleSelector, position36)
			}
			return true
		l35:
			position, tokenIndex, depth = position35, tokenIndex35, depth35
			return false
		},
		/* 5 SimpleSelectorOp <- <((&('t') ('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p')) | (&('s') ('s' 'o' 'u' 'r' 'c' 'e')) | (&('n') ('n' 'a' 'm' 'e' 's' 'p' 'a' 'c' 'e')) | (&('p') ('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')) | (&('i') ('i' 'd')) | (&('b') ('b' 'o' 'd' 'y')) | (&('*') '*'))> */
		nil,
		/* 6 CompoundSelector <- <('(' SimpleSelector (',' WSX SimpleSelector)* ')')> */
		nil,
		/* 7 FunctionSelector <- <(Function '(' SimpleSelector ')')> */
		nil,
		/* 8 Function <- <(<FunctionOp> Action6)> */
		nil,
		/* 9 FunctionOp <- <(('C' 'O' 'U' 'N' 'T') / ('M' 'I' 'N') / ('M' 'A' 'X'))> */
		nil,
		/* 10 Source <- <('F' 'R' 'O' 'M' WS Namespace Action7)> */
		func() bool {
			position46, tokenIndex46, depth46 := position, tokenIndex, depth
			{
				position47 := position
				depth++
				if buffer[position] != rune('F') {
					goto l46
				}
				position++
				if buffer[position] != rune('R') {
					goto l46
				}
				position++
				if buffer[position] != rune('O') {
					goto l46
				}
				position++
				if buffer[position] != rune('M') {
					goto l46
				}
				position++
				if !_rules[ruleWS]() {
					goto l46
				}
				{
					position48 := position
					depth++
					{
						position49, tokenIndex49, depth49 := position, tokenIndex, depth
						{
							position51 := position
							depth++
							if !_rules[ruleNamespacePart]() {
								goto l50
							}
						l52:
							{
								position53, tokenIndex53, depth53 := position, tokenIndex, depth
								if buffer[position] != rune('.') {
									goto l53
								}
								position++
								if !_rules[ruleNamespacePart]() {
									goto l53
								}
								goto l52
							l53:
								position, tokenIndex, depth = position53, tokenIndex53, depth53
							}
							{
								position54, tokenIndex54, depth54 := position, tokenIndex, depth
								if buffer[position] != rune('.') {
									goto l54
								}
								position++
								if !_rules[ruleWildcard]() {
									goto l54
								}
								goto l55
							l54:
								position, tokenIndex, depth = position54, tokenIndex54, depth54
							}
						l55:
							depth--
							add(rulePegText, position51)
						}
						goto l49
					l50:
						position, tokenIndex, depth = position49, tokenIndex49, depth49
						{
							position56 := position
							depth++
							if !_rules[ruleWildcard]() {
								goto l46
							}
							depth--
							add(rulePegText, position56)
						}
					}
				l49:
					depth--
					add(ruleNamespace, position48)
				}
				{
					add(ruleAction7, position)
				}
				depth--
				add(ruleSource, position47)
			}
			return true
		l46:
			position, tokenIndex, depth = position46, tokenIndex46, depth46
			return false
		},
		/* 11 Namespace <- <(<(NamespacePart ('.' NamespacePart)* ('.' Wildcard)?)> / <Wildcard>)> */
		nil,
		/* 12 NamespacePart <- <((&('-') '-') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position59, tokenIndex59, depth59 := position, tokenIndex, depth
			{
				position60 := position
				depth++
				{
					switch buffer[position] {
					case '-':
						if buffer[position] != rune('-') {
							goto l59
						}
						position++
						break
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l59
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l59
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l59
						}
						position++
						break
					}
				}

			l61:
				{
					position62, tokenIndex62, depth62 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '-':
							if buffer[position] != rune('-') {
								goto l62
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l62
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l62
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l62
							}
							position++
							break
						}
					}

					goto l61
				l62:
					position, tokenIndex, depth = position62, tokenIndex62, depth62
				}
				depth--
				add(ruleNamespacePart, position60)
			}
			return true
		l59:
			position, tokenIndex, depth = position59, tokenIndex59, depth59
			return false
		},
		/* 13 Wildcard <- <'*'> */
		func() bool {
			position65, tokenIndex65, depth65 := position, tokenIndex, depth
			{
				position66 := position
				depth++
				if buffer[position] != rune('*') {
					goto l65
				}
				position++
				depth--
				add(ruleWildcard, position66)
			}
			return true
		l65:
			position, tokenIndex, depth = position65, tokenIndex65, depth65
			return false
		},
		/* 14 Criteria <- <('W' 'H' 'E' 'R' 'E' WS MultiCriteria Action8)> */
		func() bool {
			position67, tokenIndex67, depth67 := position, tokenIndex, depth
			{
				position68 := position
				depth++
				if buffer[position] != rune('W') {
					goto l67
				}
				position++
				if buffer[position] != rune('H') {
					goto l67
				}
				position++
				if buffer[position] != rune('E') {
					goto l67
				}
				position++
				if buffer[position] != rune('R') {
					goto l67
				}
				position++
				if buffer[position] != rune('E') {
					goto l67
				}
				position++
				if !_rules[ruleWS]() {
					goto l67
				}
				if !_rules[ruleMultiCriteria]() {
					goto l67
				}
				{
					add(ruleAction8, position)
				}
				depth--
				add(ruleCriteria, position68)
			}
			return true
		l67:
			position, tokenIndex, depth = position67, tokenIndex67, depth67
			return false
		},
		/* 15 MultiCriteria <- <(CompoundCriteria (WS Boolean WS CompoundCriteria Action9)*)> */
		func() bool {
			position70, tokenIndex70, depth70 := position, tokenIndex, depth
			{
				position71 := position
				depth++
				if !_rules[ruleCompoundCriteria]() {
					goto l70
				}
			l72:
				{
					position73, tokenIndex73, depth73 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l73
					}
					{
						position74 := position
						depth++
						{
							position75 := position
							depth++
							{
								position76 := position
								depth++
								{
									position77, tokenIndex77, depth77 := position, tokenIndex, depth
									if buffer[position] != rune('A') {
										goto l78
									}
									position++
									if buffer[position] != rune('N') {
										goto l78
									}
									position++
									if buffer[position] != rune('D') {
										goto l78
									}
									position++
									goto l77
								l78:
									position, tokenIndex, depth = position77, tokenIndex77, depth77
									if buffer[position] != rune('O') {
										goto l73
									}
									position++
									if buffer[position] != rune('R') {
										goto l73
									}
									position++
								}
							l77:
								depth--
								add(ruleBooleanOp, position76)
							}
							depth--
							add(rulePegText, position75)
						}
						{
							add(ruleAction21, position)
						}
						depth--
						add(ruleBoolean, position74)
					}
					if !_rules[ruleWS]() {
						goto l73
					}
					if !_rules[ruleCompoundCriteria]() {
						goto l73
					}
					{
						add(ruleAction9, position)
					}
					goto l72
				l73:
					position, tokenIndex, depth = position73, tokenIndex73, depth73
				}
				depth--
				add(ruleMultiCriteria, position71)
			}
			return true
		l70:
			position, tokenIndex, depth = position70, tokenIndex70, depth70
			return false
		},
		/* 16 CompoundCriteria <- <((&('N') ('N' 'O' 'T' WS CompoundCriteria Action10)) | (&('(') ('(' MultiCriteria ')')) | (&('i' | 'p' | 's' | 't') SimpleCriteria))> */
		func() bool {
			position81, tokenIndex81, depth81 := position, tokenIndex, depth
			{
				position82 := position
				depth++
				{
					switch buffer[position] {
					case 'N':
						if buffer[position] != rune('N') {
							goto l81
						}
						position++
						if buffer[position] != rune('O') {
							goto l81
						}
						position++
						if buffer[position] != rune('T') {
							goto l81
						}
						position++
						if !_rules[ruleWS]() {
							goto l81
						}
						if !_rules[ruleCompoundCriteria]() {
							goto l81
						}
						{
							add(ruleAction10, position)
						}
						break
					case '(':
						if buffer[position] != rune('(') {
							goto l81
						}
						position++
						if !_rules[ruleMultiCriteria]() {
							goto l81
						}
						if buffer[position] != rune(')') {
							goto l81
						}
						position++
						break
					default:
						{
							position85 := position
							depth++
							{
								position86, tokenIndex86, depth86 := position, tokenIndex, depth
								{
									position88 := position
									depth++
									{
										switch buffer[position] {
										case 's':
											{
												position90 := position
												depth++
												{
													position91 := position
													depth++
													if buffer[position] != rune('s') {
														goto l87
													}
													position++
													if buffer[position] != rune('o') {
														goto l87
													}
													position++
													if buffer[position] != rune('u') {
														goto l87
													}
													position++
													if buffer[position] != rune('r') {
														goto l87
													}
													position++
													if buffer[position] != rune('c') {
														goto l87
													}
													position++
													if buffer[position] != rune('e') {
														goto l87
													}
													position++
													depth--
													add(rulePegText, position91)
												}
												{
													add(ruleAction17, position)
												}
												if !_rules[ruleWSX]() {
													goto l87
												}
												if !_rules[ruleValueCompare]() {
													goto l87
												}
												if !_rules[ruleWSX]() {
													goto l87
												}
												if !_rules[rulePublisherId]() {
													goto l87
												}
												{
													add(ruleAction18, position)
												}
												depth--
												add(ruleSourceCriteria, position90)
											}
											break
										case 'p':
											{
												position94 := position
												depth++
												{
													position95 := position
													depth++
													if buffer[position] != rune('p') {
														goto l87
													}
													position++
													if buffer[position] != rune('u') {
														goto l87
													}
													position++
													if buffer[position] != rune('b') {
														goto l87
													}
													position++
													if buffer[position] != rune('l') {
														goto l87
													}
													position++
													if buffer[position] != rune('i') {
														goto l87
													}
													position++
													if buffer[position] != rune('s') {
														goto l87
													}
													position++
													if buffer[position] != rune('h') {
														goto l87
													}
													position++
													if buffer[position] != rune('e') {
														goto l87
													}
													position++
													if buffer[position] != rune('r') {
														goto l87
													}
													position++
													depth--
													add(rulePegText, position95)
												}
												{
													add(ruleAction15, position)
												}
												if !_rules[ruleWSX]() {
													goto l87
												}
												if !_rules[ruleValueCompare]() {
													goto l87
												}
												if !_rules[ruleWSX]() {
													goto l87
												}
												if !_rules[rulePublisherId]() {
													goto l87
												}
												{
													add(ruleAction16, position)
												}
												depth--
												add(rulePublisherCriteria, position94)
											}
											break
										default:
											{
												position98 := position
												depth++
												{
													position99 := position
													depth++
													if buffer[position] != rune('i') {
														goto l87
													}
													position++
													if buffer[position] != rune('d') {
														goto l87
													}
													position++
													depth--
													add(rulePegText, position99)
												}
												{
													add(ruleAction13, position)
												}
												if !_rules[ruleWSX]() {
													goto l87
												}
												if !_rules[ruleValueCompare]() {
													goto l87
												}
												if !_rules[ruleWSX]() {
													goto l87
												}
												{
													position101 := position
													depth++
													{
														position102 := position
														depth++
														{
															switch buffer[position] {
															case ':':
																if buffer[position] != rune(':') {
																	goto l87
																}
																position++
																break
															case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
																if c := buffer[position]; c < rune('0') || c > rune('9') {
																	goto l87
																}
																position++
																break
															case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
																if c := buffer[position]; c < rune('A') || c > rune('Z') {
																	goto l87
																}
																position++
																break
															default:
																if c := buffer[position]; c < rune('a') || c > rune('z') {
																	goto l87
																}
																position++
																break
															}
														}

													l103:
														{
															position104, tokenIndex104, depth104 := position, tokenIndex, depth
															{
																switch buffer[position] {
																case ':':
																	if buffer[position] != rune(':') {
																		goto l104
																	}
																	position++
																	break
																case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
																	if c := buffer[position]; c < rune('0') || c > rune('9') {
																		goto l104
																	}
																	position++
																	break
																case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
																	if c := buffer[position]; c < rune('A') || c > rune('Z') {
																		goto l104
																	}
																	position++
																	break
																default:
																	if c := buffer[position]; c < rune('a') || c > rune('z') {
																		goto l104
																	}
																	position++
																	break
																}
															}

															goto l103
														l104:
															position, tokenIndex, depth = position104, tokenIndex104, depth104
														}
														depth--
														add(rulePegText, position102)
													}
													depth--
													add(ruleStatementId, position101)
												}
												{
													add(ruleAction14, position)
												}
												depth--
												add(ruleIdCriteria, position98)
											}
											break
										}
									}

									depth--
									add(ruleValueCriteria, position88)
								}
								{
									add(ruleAction11, position)
								}
								goto l86
							l87:
								position, tokenIndex, depth = position86, tokenIndex86, depth86
								{
									position109 := position
									depth++
									if buffer[position] != rune('t') {
										goto l81
									}
									position++
									if buffer[position] != rune('i') {
										goto l81
									}
									position++
									if buffer[position] != rune('m') {
										goto l81
									}
									position++
									if buffer[position] != rune('e') {
										goto l81
									}
									position++
									if buffer[position] != rune('s') {
										goto l81
									}
									position++
									if buffer[position] != rune('t') {
										goto l81
									}
									position++
									if buffer[position] != rune('a') {
										goto l81
									}
									position++
									if buffer[position] != rune('m') {
										goto l81
									}
									position++
									if buffer[position] != rune('p') {
										goto l81
									}
									position++
									if !_rules[ruleWSX]() {
										goto l81
									}
									{
										position110 := position
										depth++
										{
											position111 := position
											depth++
											{
												position112 := position
												depth++
												{
													position113, tokenIndex113, depth113 := position, tokenIndex, depth
													if buffer[position] != rune('<') {
														goto l114
													}
													position++
													if buffer[position] != rune('=') {
														goto l114
													}
													position++
													goto l113
												l114:
													position, tokenIndex, depth = position113, tokenIndex113, depth113
													if buffer[position] != rune('>') {
														goto l115
													}
													position++
													if buffer[position] != rune('=') {
														goto l115
													}
													position++
													goto l113
												l115:
													position, tokenIndex, depth = position113, tokenIndex113, depth113
													{
														switch buffer[position] {
														case '>':
															if buffer[position] != rune('>') {
																goto l81
															}
															position++
															break
														case '!':
															if buffer[position] != rune('!') {
																goto l81
															}
															position++
															if buffer[position] != rune('=') {
																goto l81
															}
															position++
															break
														case '=':
															if buffer[position] != rune('=') {
																goto l81
															}
															position++
															break
														default:
															if buffer[position] != rune('<') {
																goto l81
															}
															position++
															break
														}
													}

												}
											l113:
												depth--
												add(ruleComparisonOp, position112)
											}
											depth--
											add(rulePegText, position111)
										}
										{
											add(ruleAction22, position)
										}
										depth--
										add(ruleComparison, position110)
									}
									if !_rules[ruleWSX]() {
										goto l81
									}
									if !_rules[ruleUInt]() {
										goto l81
									}
									{
										add(ruleAction20, position)
									}
									depth--
									add(ruleTimeCriteria, position109)
								}
								{
									add(ruleAction12, position)
								}
							}
						l86:
							depth--
							add(ruleSimpleCriteria, position85)
						}
						break
					}
				}

				depth--
				add(ruleCompoundCriteria, position82)
			}
			return true
		l81:
			position, tokenIndex, depth = position81, tokenIndex81, depth81
			return false
		},
		/* 17 SimpleCriteria <- <((ValueCriteria Action11) / (TimeCriteria Action12))> */
		nil,
		/* 18 ValueCriteria <- <((&('s') SourceCriteria) | (&('p') PublisherCriteria) | (&('i') IdCriteria))> */
		nil,
		/* 19 IdCriteria <- <(<('i' 'd')> Action13 WSX ValueCompare WSX StatementId Action14)> */
		nil,
		/* 20 PublisherCriteria <- <(<('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')> Action15 WSX ValueCompare WSX PublisherId Action16)> */
		nil,
		/* 21 SourceCriteria <- <(<('s' 'o' 'u' 'r' 'c' 'e')> Action17 WSX ValueCompare WSX PublisherId Action18)> */
		nil,
		/* 22 ValueCompare <- <(<ValueCompareOp> Action19)> */
		func() bool {
			position125, tokenIndex125, depth125 := position, tokenIndex, depth
			{
				position126 := position
				depth++
				{
					position127 := position
					depth++
					{
						position128 := position
						depth++
						{
							position129, tokenIndex129, depth129 := position, tokenIndex, depth
							if buffer[position] != rune('=') {
								goto l130
							}
							position++
							goto l129
						l130:
							position, tokenIndex, depth = position129, tokenIndex129, depth129
							if buffer[position] != rune('!') {
								goto l125
							}
							position++
							if buffer[position] != rune('=') {
								goto l125
							}
							position++
						}
					l129:
						depth--
						add(ruleValueCompareOp, position128)
					}
					depth--
					add(rulePegText, position127)
				}
				{
					add(ruleAction19, position)
				}
				depth--
				add(ruleValueCompare, position126)
			}
			return true
		l125:
			position, tokenIndex, depth = position125, tokenIndex125, depth125
			return false
		},
		/* 23 ValueCompareOp <- <('=' / ('!' '='))> */
		nil,
		/* 24 TimeCriteria <- <('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p' WSX Comparison WSX UInt Action20)> */
		nil,
		/* 25 Boolean <- <(<BooleanOp> Action21)> */
		nil,
		/* 26 BooleanOp <- <(('A' 'N' 'D') / ('O' 'R'))> */
		nil,
		/* 27 Comparison <- <(<ComparisonOp> Action22)> */
		nil,
		/* 28 ComparisonOp <- <(('<' '=') / ('>' '=') / ((&('>') '>') | (&('!') ('!' '=')) | (&('=') '=') | (&('<') '<')))> */
		nil,
		/* 29 Limit <- <('L' 'I' 'M' 'I' 'T' WS UInt Action23)> */
		nil,
		/* 30 StatementId <- <<((&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 31 PublisherId <- <<((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		func() bool {
			position140, tokenIndex140, depth140 := position, tokenIndex, depth
			{
				position141 := position
				depth++
				{
					position142 := position
					depth++
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l140
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l140
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l140
							}
							position++
							break
						}
					}

				l143:
					{
						position144, tokenIndex144, depth144 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l144
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l144
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l144
								}
								position++
								break
							}
						}

						goto l143
					l144:
						position, tokenIndex, depth = position144, tokenIndex144, depth144
					}
					depth--
					add(rulePegText, position142)
				}
				depth--
				add(rulePublisherId, position141)
			}
			return true
		l140:
			position, tokenIndex, depth = position140, tokenIndex140, depth140
			return false
		},
		/* 32 UInt <- <<[0-9]+>> */
		func() bool {
			position147, tokenIndex147, depth147 := position, tokenIndex, depth
			{
				position148 := position
				depth++
				{
					position149 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l147
					}
					position++
				l150:
					{
						position151, tokenIndex151, depth151 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l151
						}
						position++
						goto l150
					l151:
						position, tokenIndex, depth = position151, tokenIndex151, depth151
					}
					depth--
					add(rulePegText, position149)
				}
				depth--
				add(ruleUInt, position148)
			}
			return true
		l147:
			position, tokenIndex, depth = position147, tokenIndex147, depth147
			return false
		},
		/* 33 WS <- <WhiteSpace+> */
		func() bool {
			position152, tokenIndex152, depth152 := position, tokenIndex, depth
			{
				position153 := position
				depth++
				if !_rules[ruleWhiteSpace]() {
					goto l152
				}
			l154:
				{
					position155, tokenIndex155, depth155 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l155
					}
					goto l154
				l155:
					position, tokenIndex, depth = position155, tokenIndex155, depth155
				}
				depth--
				add(ruleWS, position153)
			}
			return true
		l152:
			position, tokenIndex, depth = position152, tokenIndex152, depth152
			return false
		},
		/* 34 WSX <- <WhiteSpace*> */
		func() bool {
			{
				position157 := position
				depth++
			l158:
				{
					position159, tokenIndex159, depth159 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l159
					}
					goto l158
				l159:
					position, tokenIndex, depth = position159, tokenIndex159, depth159
				}
				depth--
				add(ruleWSX, position157)
			}
			return true
		},
		/* 35 WhiteSpace <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') EOL))> */
		func() bool {
			position160, tokenIndex160, depth160 := position, tokenIndex, depth
			{
				position161 := position
				depth++
				{
					switch buffer[position] {
					case '\t':
						if buffer[position] != rune('\t') {
							goto l160
						}
						position++
						break
					case ' ':
						if buffer[position] != rune(' ') {
							goto l160
						}
						position++
						break
					default:
						{
							position163 := position
							depth++
							{
								position164, tokenIndex164, depth164 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l165
								}
								position++
								if buffer[position] != rune('\n') {
									goto l165
								}
								position++
								goto l164
							l165:
								position, tokenIndex, depth = position164, tokenIndex164, depth164
								if buffer[position] != rune('\n') {
									goto l166
								}
								position++
								goto l164
							l166:
								position, tokenIndex, depth = position164, tokenIndex164, depth164
								if buffer[position] != rune('\r') {
									goto l160
								}
								position++
							}
						l164:
							depth--
							add(ruleEOL, position163)
						}
						break
					}
				}

				depth--
				add(ruleWhiteSpace, position161)
			}
			return true
		l160:
			position, tokenIndex, depth = position160, tokenIndex160, depth160
			return false
		},
		/* 36 EOL <- <(('\r' '\n') / '\n' / '\r')> */
		nil,
		/* 37 EOF <- <!.> */
		func() bool {
			position168, tokenIndex168, depth168 := position, tokenIndex, depth
			{
				position169 := position
				depth++
				{
					position170, tokenIndex170, depth170 := position, tokenIndex, depth
					if !matchDot() {
						goto l170
					}
					goto l168
				l170:
					position, tokenIndex, depth = position170, tokenIndex170, depth170
				}
				depth--
				add(ruleEOF, position169)
			}
			return true
		l168:
			position, tokenIndex, depth = position168, tokenIndex168, depth168
			return false
		},
		/* 39 Action0 <- <{ p.setSelectOp() }> */
		nil,
		/* 40 Action1 <- <{ p.setDeleteOp() }> */
		nil,
		/* 41 Action2 <- <{ p.setSimpleSelector() }> */
		nil,
		/* 42 Action3 <- <{ p.setCompoundSelector() }> */
		nil,
		/* 43 Action4 <- <{ p.setFunctionSelector() }> */
		nil,
		nil,
		/* 45 Action5 <- <{ p.push(text) }> */
		nil,
		/* 46 Action6 <- <{ p.push(text) }> */
		nil,
		/* 47 Action7 <- <{ p.setNamespace(text) }> */
		nil,
		/* 48 Action8 <- <{ p.setCriteria() }> */
		nil,
		/* 49 Action9 <- <{ p.addCompoundCriteria() }> */
		nil,
		/* 50 Action10 <- <{ p.addNegatedCriteria() }> */
		nil,
		/* 51 Action11 <- <{ p.addValueCriteria() }> */
		nil,
		/* 52 Action12 <- <{ p.addTimeCriteria() }> */
		nil,
		/* 53 Action13 <- <{ p.push(text) }> */
		nil,
		/* 54 Action14 <- <{ p.push(text) }> */
		nil,
		/* 55 Action15 <- <{ p.push(text) }> */
		nil,
		/* 56 Action16 <- <{ p.push(text) }> */
		nil,
		/* 57 Action17 <- <{ p.push(text) }> */
		nil,
		/* 58 Action18 <- <{ p.push(text) }> */
		nil,
		/* 59 Action19 <- <{ p.push(text) }> */
		nil,
		/* 60 Action20 <- <{ p.push(text) }> */
		nil,
		/* 61 Action21 <- <{ p.push(text) }> */
		nil,
		/* 62 Action22 <- <{ p.push(text) }> */
		nil,
		/* 63 Action23 <- <{ p.setLimit(text) }> */
		nil,
	}
	p.rules = _rules
}
