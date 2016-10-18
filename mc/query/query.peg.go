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
	ruleRangeCriteria
	ruleRangeSelector
	ruleRangeSelectorOp
	ruleBoolean
	ruleBooleanOp
	ruleComparison
	ruleComparisonOp
	ruleIndexCriteria
	ruleWKICriteria
	ruleOrder
	ruleOrderSpec
	ruleOrderSelectorSpec
	ruleOrderSelector
	ruleOrderSelectorOp
	ruleOrderDir
	ruleOrderDirOp
	ruleLimit
	ruleStatementId
	rulePublisherId
	ruleWKI
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
	ruleAction24
	ruleAction25
	ruleAction26
	ruleAction27
	ruleAction28
	ruleAction29
	ruleAction30
	ruleAction31
	ruleAction32

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
	"RangeCriteria",
	"RangeSelector",
	"RangeSelectorOp",
	"Boolean",
	"BooleanOp",
	"Comparison",
	"ComparisonOp",
	"IndexCriteria",
	"WKICriteria",
	"Order",
	"OrderSpec",
	"OrderSelectorSpec",
	"OrderSelector",
	"OrderSelectorOp",
	"OrderDir",
	"OrderDirOp",
	"Limit",
	"StatementId",
	"PublisherId",
	"WKI",
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
	"Action24",
	"Action25",
	"Action26",
	"Action27",
	"Action28",
	"Action29",
	"Action30",
	"Action31",
	"Action32",

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
	rules  [85]func() bool
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
			p.addRangeCriteria()
		case ruleAction13:
			p.addIndexCriteria()
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
			p.push(text)
		case ruleAction24:
			p.push(text)
		case ruleAction25:
			p.push(text)
		case ruleAction26:
			p.push(text)
		case ruleAction27:
			p.setOrder()
		case ruleAction28:
			p.addOrderSelector()
		case ruleAction29:
			p.setOrderDir()
		case ruleAction30:
			p.push(text)
		case ruleAction31:
			p.push(text)
		case ruleAction32:
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
								if buffer[position] != rune('O') {
									goto l23
								}
								position++
								if buffer[position] != rune('R') {
									goto l23
								}
								position++
								if buffer[position] != rune('D') {
									goto l23
								}
								position++
								if buffer[position] != rune('E') {
									goto l23
								}
								position++
								if buffer[position] != rune('R') {
									goto l23
								}
								position++
								if !_rules[ruleWS]() {
									goto l23
								}
								if buffer[position] != rune('B') {
									goto l23
								}
								position++
								if buffer[position] != rune('Y') {
									goto l23
								}
								position++
								if !_rules[ruleWS]() {
									goto l23
								}
								{
									position26 := position
									depth++
									if !_rules[ruleOrderSelectorSpec]() {
										goto l23
									}
								l27:
									{
										position28, tokenIndex28, depth28 := position, tokenIndex, depth
										if buffer[position] != rune(',') {
											goto l28
										}
										position++
										if !_rules[ruleWSX]() {
											goto l28
										}
										if !_rules[ruleOrderSelectorSpec]() {
											goto l28
										}
										goto l27
									l28:
										position, tokenIndex, depth = position28, tokenIndex28, depth28
									}
									depth--
									add(ruleOrderSpec, position26)
								}
								{
									add(ruleAction27, position)
								}
								depth--
								add(ruleOrder, position25)
							}
							goto l24
						l23:
							position, tokenIndex, depth = position23, tokenIndex23, depth23
						}
					l24:
						{
							position30, tokenIndex30, depth30 := position, tokenIndex, depth
							if !_rules[ruleWS]() {
								goto l30
							}
							{
								position32 := position
								depth++
								if buffer[position] != rune('L') {
									goto l30
								}
								position++
								if buffer[position] != rune('I') {
									goto l30
								}
								position++
								if buffer[position] != rune('M') {
									goto l30
								}
								position++
								if buffer[position] != rune('I') {
									goto l30
								}
								position++
								if buffer[position] != rune('T') {
									goto l30
								}
								position++
								if !_rules[ruleWS]() {
									goto l30
								}
								if !_rules[ruleUInt]() {
									goto l30
								}
								{
									add(ruleAction32, position)
								}
								depth--
								add(ruleLimit, position32)
							}
							goto l31
						l30:
							position, tokenIndex, depth = position30, tokenIndex30, depth30
						}
					l31:
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
						position35 := position
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
							position36, tokenIndex36, depth36 := position, tokenIndex, depth
							if !_rules[ruleWS]() {
								goto l36
							}
							if !_rules[ruleCriteria]() {
								goto l36
							}
							goto l37
						l36:
							position, tokenIndex, depth = position36, tokenIndex36, depth36
						}
					l37:
						depth--
						add(ruleDelete, position35)
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
		/* 1 Select <- <('S' 'E' 'L' 'E' 'C' 'T' WS Selector WS Source (WS Criteria)? (WS Order)? (WS Limit)?)> */
		nil,
		/* 2 Delete <- <('D' 'E' 'L' 'E' 'T' 'E' WS Source (WS Criteria)?)> */
		nil,
		/* 3 Selector <- <((&('C' | 'M') (FunctionSelector Action4)) | (&('(') (CompoundSelector Action3)) | (&('*' | 'b' | 'c' | 'i' | 'n' | 'p' | 's' | 't') (SimpleSelector Action2)))> */
		nil,
		/* 4 SimpleSelector <- <(<SimpleSelectorOp> Action5)> */
		func() bool {
			position42, tokenIndex42, depth42 := position, tokenIndex, depth
			{
				position43 := position
				depth++
				{
					position44 := position
					depth++
					{
						position45 := position
						depth++
						{
							switch buffer[position] {
							case 'c':
								if buffer[position] != rune('c') {
									goto l42
								}
								position++
								if buffer[position] != rune('o') {
									goto l42
								}
								position++
								if buffer[position] != rune('u') {
									goto l42
								}
								position++
								if buffer[position] != rune('n') {
									goto l42
								}
								position++
								if buffer[position] != rune('t') {
									goto l42
								}
								position++
								if buffer[position] != rune('e') {
									goto l42
								}
								position++
								if buffer[position] != rune('r') {
									goto l42
								}
								position++
								break
							case 't':
								if buffer[position] != rune('t') {
									goto l42
								}
								position++
								if buffer[position] != rune('i') {
									goto l42
								}
								position++
								if buffer[position] != rune('m') {
									goto l42
								}
								position++
								if buffer[position] != rune('e') {
									goto l42
								}
								position++
								if buffer[position] != rune('s') {
									goto l42
								}
								position++
								if buffer[position] != rune('t') {
									goto l42
								}
								position++
								if buffer[position] != rune('a') {
									goto l42
								}
								position++
								if buffer[position] != rune('m') {
									goto l42
								}
								position++
								if buffer[position] != rune('p') {
									goto l42
								}
								position++
								break
							case 's':
								if buffer[position] != rune('s') {
									goto l42
								}
								position++
								if buffer[position] != rune('o') {
									goto l42
								}
								position++
								if buffer[position] != rune('u') {
									goto l42
								}
								position++
								if buffer[position] != rune('r') {
									goto l42
								}
								position++
								if buffer[position] != rune('c') {
									goto l42
								}
								position++
								if buffer[position] != rune('e') {
									goto l42
								}
								position++
								break
							case 'n':
								if buffer[position] != rune('n') {
									goto l42
								}
								position++
								if buffer[position] != rune('a') {
									goto l42
								}
								position++
								if buffer[position] != rune('m') {
									goto l42
								}
								position++
								if buffer[position] != rune('e') {
									goto l42
								}
								position++
								if buffer[position] != rune('s') {
									goto l42
								}
								position++
								if buffer[position] != rune('p') {
									goto l42
								}
								position++
								if buffer[position] != rune('a') {
									goto l42
								}
								position++
								if buffer[position] != rune('c') {
									goto l42
								}
								position++
								if buffer[position] != rune('e') {
									goto l42
								}
								position++
								break
							case 'p':
								if buffer[position] != rune('p') {
									goto l42
								}
								position++
								if buffer[position] != rune('u') {
									goto l42
								}
								position++
								if buffer[position] != rune('b') {
									goto l42
								}
								position++
								if buffer[position] != rune('l') {
									goto l42
								}
								position++
								if buffer[position] != rune('i') {
									goto l42
								}
								position++
								if buffer[position] != rune('s') {
									goto l42
								}
								position++
								if buffer[position] != rune('h') {
									goto l42
								}
								position++
								if buffer[position] != rune('e') {
									goto l42
								}
								position++
								if buffer[position] != rune('r') {
									goto l42
								}
								position++
								break
							case 'i':
								if buffer[position] != rune('i') {
									goto l42
								}
								position++
								if buffer[position] != rune('d') {
									goto l42
								}
								position++
								break
							case 'b':
								if buffer[position] != rune('b') {
									goto l42
								}
								position++
								if buffer[position] != rune('o') {
									goto l42
								}
								position++
								if buffer[position] != rune('d') {
									goto l42
								}
								position++
								if buffer[position] != rune('y') {
									goto l42
								}
								position++
								break
							default:
								if buffer[position] != rune('*') {
									goto l42
								}
								position++
								break
							}
						}

						depth--
						add(ruleSimpleSelectorOp, position45)
					}
					depth--
					add(rulePegText, position44)
				}
				{
					add(ruleAction5, position)
				}
				depth--
				add(ruleSimpleSelector, position43)
			}
			return true
		l42:
			position, tokenIndex, depth = position42, tokenIndex42, depth42
			return false
		},
		/* 5 SimpleSelectorOp <- <((&('c') ('c' 'o' 'u' 'n' 't' 'e' 'r')) | (&('t') ('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p')) | (&('s') ('s' 'o' 'u' 'r' 'c' 'e')) | (&('n') ('n' 'a' 'm' 'e' 's' 'p' 'a' 'c' 'e')) | (&('p') ('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')) | (&('i') ('i' 'd')) | (&('b') ('b' 'o' 'd' 'y')) | (&('*') '*'))> */
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
			position53, tokenIndex53, depth53 := position, tokenIndex, depth
			{
				position54 := position
				depth++
				if buffer[position] != rune('F') {
					goto l53
				}
				position++
				if buffer[position] != rune('R') {
					goto l53
				}
				position++
				if buffer[position] != rune('O') {
					goto l53
				}
				position++
				if buffer[position] != rune('M') {
					goto l53
				}
				position++
				if !_rules[ruleWS]() {
					goto l53
				}
				{
					position55 := position
					depth++
					{
						position56, tokenIndex56, depth56 := position, tokenIndex, depth
						{
							position58 := position
							depth++
							if !_rules[ruleNamespacePart]() {
								goto l57
							}
						l59:
							{
								position60, tokenIndex60, depth60 := position, tokenIndex, depth
								if buffer[position] != rune('.') {
									goto l60
								}
								position++
								if !_rules[ruleNamespacePart]() {
									goto l60
								}
								goto l59
							l60:
								position, tokenIndex, depth = position60, tokenIndex60, depth60
							}
							{
								position61, tokenIndex61, depth61 := position, tokenIndex, depth
								if buffer[position] != rune('.') {
									goto l61
								}
								position++
								if !_rules[ruleWildcard]() {
									goto l61
								}
								goto l62
							l61:
								position, tokenIndex, depth = position61, tokenIndex61, depth61
							}
						l62:
							depth--
							add(rulePegText, position58)
						}
						goto l56
					l57:
						position, tokenIndex, depth = position56, tokenIndex56, depth56
						{
							position63 := position
							depth++
							if !_rules[ruleWildcard]() {
								goto l53
							}
							depth--
							add(rulePegText, position63)
						}
					}
				l56:
					depth--
					add(ruleNamespace, position55)
				}
				{
					add(ruleAction7, position)
				}
				depth--
				add(ruleSource, position54)
			}
			return true
		l53:
			position, tokenIndex, depth = position53, tokenIndex53, depth53
			return false
		},
		/* 11 Namespace <- <(<(NamespacePart ('.' NamespacePart)* ('.' Wildcard)?)> / <Wildcard>)> */
		nil,
		/* 12 NamespacePart <- <((&('-') '-') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position66, tokenIndex66, depth66 := position, tokenIndex, depth
			{
				position67 := position
				depth++
				{
					switch buffer[position] {
					case '-':
						if buffer[position] != rune('-') {
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

			l68:
				{
					position69, tokenIndex69, depth69 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '-':
							if buffer[position] != rune('-') {
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
				add(ruleNamespacePart, position67)
			}
			return true
		l66:
			position, tokenIndex, depth = position66, tokenIndex66, depth66
			return false
		},
		/* 13 Wildcard <- <'*'> */
		func() bool {
			position72, tokenIndex72, depth72 := position, tokenIndex, depth
			{
				position73 := position
				depth++
				if buffer[position] != rune('*') {
					goto l72
				}
				position++
				depth--
				add(ruleWildcard, position73)
			}
			return true
		l72:
			position, tokenIndex, depth = position72, tokenIndex72, depth72
			return false
		},
		/* 14 Criteria <- <('W' 'H' 'E' 'R' 'E' WS MultiCriteria Action8)> */
		func() bool {
			position74, tokenIndex74, depth74 := position, tokenIndex, depth
			{
				position75 := position
				depth++
				if buffer[position] != rune('W') {
					goto l74
				}
				position++
				if buffer[position] != rune('H') {
					goto l74
				}
				position++
				if buffer[position] != rune('E') {
					goto l74
				}
				position++
				if buffer[position] != rune('R') {
					goto l74
				}
				position++
				if buffer[position] != rune('E') {
					goto l74
				}
				position++
				if !_rules[ruleWS]() {
					goto l74
				}
				if !_rules[ruleMultiCriteria]() {
					goto l74
				}
				{
					add(ruleAction8, position)
				}
				depth--
				add(ruleCriteria, position75)
			}
			return true
		l74:
			position, tokenIndex, depth = position74, tokenIndex74, depth74
			return false
		},
		/* 15 MultiCriteria <- <(CompoundCriteria (WS Boolean WS CompoundCriteria Action9)*)> */
		func() bool {
			position77, tokenIndex77, depth77 := position, tokenIndex, depth
			{
				position78 := position
				depth++
				if !_rules[ruleCompoundCriteria]() {
					goto l77
				}
			l79:
				{
					position80, tokenIndex80, depth80 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l80
					}
					{
						position81 := position
						depth++
						{
							position82 := position
							depth++
							{
								position83 := position
								depth++
								{
									position84, tokenIndex84, depth84 := position, tokenIndex, depth
									if buffer[position] != rune('A') {
										goto l85
									}
									position++
									if buffer[position] != rune('N') {
										goto l85
									}
									position++
									if buffer[position] != rune('D') {
										goto l85
									}
									position++
									goto l84
								l85:
									position, tokenIndex, depth = position84, tokenIndex84, depth84
									if buffer[position] != rune('O') {
										goto l80
									}
									position++
									if buffer[position] != rune('R') {
										goto l80
									}
									position++
								}
							l84:
								depth--
								add(ruleBooleanOp, position83)
							}
							depth--
							add(rulePegText, position82)
						}
						{
							add(ruleAction23, position)
						}
						depth--
						add(ruleBoolean, position81)
					}
					if !_rules[ruleWS]() {
						goto l80
					}
					if !_rules[ruleCompoundCriteria]() {
						goto l80
					}
					{
						add(ruleAction9, position)
					}
					goto l79
				l80:
					position, tokenIndex, depth = position80, tokenIndex80, depth80
				}
				depth--
				add(ruleMultiCriteria, position78)
			}
			return true
		l77:
			position, tokenIndex, depth = position77, tokenIndex77, depth77
			return false
		},
		/* 16 CompoundCriteria <- <((&('N') ('N' 'O' 'T' WS CompoundCriteria Action10)) | (&('(') ('(' MultiCriteria ')')) | (&('c' | 'i' | 'p' | 's' | 't' | 'w') SimpleCriteria))> */
		func() bool {
			position88, tokenIndex88, depth88 := position, tokenIndex, depth
			{
				position89 := position
				depth++
				{
					switch buffer[position] {
					case 'N':
						if buffer[position] != rune('N') {
							goto l88
						}
						position++
						if buffer[position] != rune('O') {
							goto l88
						}
						position++
						if buffer[position] != rune('T') {
							goto l88
						}
						position++
						if !_rules[ruleWS]() {
							goto l88
						}
						if !_rules[ruleCompoundCriteria]() {
							goto l88
						}
						{
							add(ruleAction10, position)
						}
						break
					case '(':
						if buffer[position] != rune('(') {
							goto l88
						}
						position++
						if !_rules[ruleMultiCriteria]() {
							goto l88
						}
						if buffer[position] != rune(')') {
							goto l88
						}
						position++
						break
					default:
						{
							position92 := position
							depth++
							{
								switch buffer[position] {
								case 'w':
									{
										position94 := position
										depth++
										{
											position95 := position
											depth++
											{
												position96 := position
												depth++
												if buffer[position] != rune('w') {
													goto l88
												}
												position++
												if buffer[position] != rune('k') {
													goto l88
												}
												position++
												if buffer[position] != rune('i') {
													goto l88
												}
												position++
												depth--
												add(rulePegText, position96)
											}
											{
												add(ruleAction25, position)
											}
											if !_rules[ruleWSX]() {
												goto l88
											}
											if buffer[position] != rune('=') {
												goto l88
											}
											position++
											if !_rules[ruleWSX]() {
												goto l88
											}
											{
												position98 := position
												depth++
												{
													position99 := position
													depth++
													{
														switch buffer[position] {
														case '_':
															if buffer[position] != rune('_') {
																goto l88
															}
															position++
															break
														case ':':
															if buffer[position] != rune(':') {
																goto l88
															}
															position++
															break
														case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
															if c := buffer[position]; c < rune('0') || c > rune('9') {
																goto l88
															}
															position++
															break
														case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
															if c := buffer[position]; c < rune('A') || c > rune('Z') {
																goto l88
															}
															position++
															break
														case '-':
															if buffer[position] != rune('-') {
																goto l88
															}
															position++
															break
														default:
															if c := buffer[position]; c < rune('a') || c > rune('z') {
																goto l88
															}
															position++
															break
														}
													}

												l100:
													{
														position101, tokenIndex101, depth101 := position, tokenIndex, depth
														{
															switch buffer[position] {
															case '_':
																if buffer[position] != rune('_') {
																	goto l101
																}
																position++
																break
															case ':':
																if buffer[position] != rune(':') {
																	goto l101
																}
																position++
																break
															case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
																if c := buffer[position]; c < rune('0') || c > rune('9') {
																	goto l101
																}
																position++
																break
															case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
																if c := buffer[position]; c < rune('A') || c > rune('Z') {
																	goto l101
																}
																position++
																break
															case '-':
																if buffer[position] != rune('-') {
																	goto l101
																}
																position++
																break
															default:
																if c := buffer[position]; c < rune('a') || c > rune('z') {
																	goto l101
																}
																position++
																break
															}
														}

														goto l100
													l101:
														position, tokenIndex, depth = position101, tokenIndex101, depth101
													}
													depth--
													add(rulePegText, position99)
												}
												depth--
												add(ruleWKI, position98)
											}
											{
												add(ruleAction26, position)
											}
											depth--
											add(ruleWKICriteria, position95)
										}
										depth--
										add(ruleIndexCriteria, position94)
									}
									{
										add(ruleAction13, position)
									}
									break
								case 'c', 't':
									{
										position106 := position
										depth++
										{
											position107 := position
											depth++
											{
												position108 := position
												depth++
												{
													position109 := position
													depth++
													{
														position110, tokenIndex110, depth110 := position, tokenIndex, depth
														if buffer[position] != rune('t') {
															goto l111
														}
														position++
														if buffer[position] != rune('i') {
															goto l111
														}
														position++
														if buffer[position] != rune('m') {
															goto l111
														}
														position++
														if buffer[position] != rune('e') {
															goto l111
														}
														position++
														if buffer[position] != rune('s') {
															goto l111
														}
														position++
														if buffer[position] != rune('t') {
															goto l111
														}
														position++
														if buffer[position] != rune('a') {
															goto l111
														}
														position++
														if buffer[position] != rune('m') {
															goto l111
														}
														position++
														if buffer[position] != rune('p') {
															goto l111
														}
														position++
														goto l110
													l111:
														position, tokenIndex, depth = position110, tokenIndex110, depth110
														if buffer[position] != rune('c') {
															goto l88
														}
														position++
														if buffer[position] != rune('o') {
															goto l88
														}
														position++
														if buffer[position] != rune('u') {
															goto l88
														}
														position++
														if buffer[position] != rune('n') {
															goto l88
														}
														position++
														if buffer[position] != rune('t') {
															goto l88
														}
														position++
														if buffer[position] != rune('e') {
															goto l88
														}
														position++
														if buffer[position] != rune('r') {
															goto l88
														}
														position++
													}
												l110:
													depth--
													add(ruleRangeSelectorOp, position109)
												}
												depth--
												add(rulePegText, position108)
											}
											{
												add(ruleAction22, position)
											}
											depth--
											add(ruleRangeSelector, position107)
										}
										if !_rules[ruleWSX]() {
											goto l88
										}
										{
											position113 := position
											depth++
											{
												position114 := position
												depth++
												{
													position115 := position
													depth++
													{
														position116, tokenIndex116, depth116 := position, tokenIndex, depth
														if buffer[position] != rune('<') {
															goto l117
														}
														position++
														if buffer[position] != rune('=') {
															goto l117
														}
														position++
														goto l116
													l117:
														position, tokenIndex, depth = position116, tokenIndex116, depth116
														if buffer[position] != rune('>') {
															goto l118
														}
														position++
														if buffer[position] != rune('=') {
															goto l118
														}
														position++
														goto l116
													l118:
														position, tokenIndex, depth = position116, tokenIndex116, depth116
														{
															switch buffer[position] {
															case '>':
																if buffer[position] != rune('>') {
																	goto l88
																}
																position++
																break
															case '!':
																if buffer[position] != rune('!') {
																	goto l88
																}
																position++
																if buffer[position] != rune('=') {
																	goto l88
																}
																position++
																break
															case '=':
																if buffer[position] != rune('=') {
																	goto l88
																}
																position++
																break
															default:
																if buffer[position] != rune('<') {
																	goto l88
																}
																position++
																break
															}
														}

													}
												l116:
													depth--
													add(ruleComparisonOp, position115)
												}
												depth--
												add(rulePegText, position114)
											}
											{
												add(ruleAction24, position)
											}
											depth--
											add(ruleComparison, position113)
										}
										if !_rules[ruleWSX]() {
											goto l88
										}
										if !_rules[ruleUInt]() {
											goto l88
										}
										{
											add(ruleAction21, position)
										}
										depth--
										add(ruleRangeCriteria, position106)
									}
									{
										add(ruleAction12, position)
									}
									break
								default:
									{
										position123 := position
										depth++
										{
											switch buffer[position] {
											case 's':
												{
													position125 := position
													depth++
													{
														position126 := position
														depth++
														if buffer[position] != rune('s') {
															goto l88
														}
														position++
														if buffer[position] != rune('o') {
															goto l88
														}
														position++
														if buffer[position] != rune('u') {
															goto l88
														}
														position++
														if buffer[position] != rune('r') {
															goto l88
														}
														position++
														if buffer[position] != rune('c') {
															goto l88
														}
														position++
														if buffer[position] != rune('e') {
															goto l88
														}
														position++
														depth--
														add(rulePegText, position126)
													}
													{
														add(ruleAction18, position)
													}
													if !_rules[ruleWSX]() {
														goto l88
													}
													if !_rules[ruleValueCompare]() {
														goto l88
													}
													if !_rules[ruleWSX]() {
														goto l88
													}
													if !_rules[rulePublisherId]() {
														goto l88
													}
													{
														add(ruleAction19, position)
													}
													depth--
													add(ruleSourceCriteria, position125)
												}
												break
											case 'p':
												{
													position129 := position
													depth++
													{
														position130 := position
														depth++
														if buffer[position] != rune('p') {
															goto l88
														}
														position++
														if buffer[position] != rune('u') {
															goto l88
														}
														position++
														if buffer[position] != rune('b') {
															goto l88
														}
														position++
														if buffer[position] != rune('l') {
															goto l88
														}
														position++
														if buffer[position] != rune('i') {
															goto l88
														}
														position++
														if buffer[position] != rune('s') {
															goto l88
														}
														position++
														if buffer[position] != rune('h') {
															goto l88
														}
														position++
														if buffer[position] != rune('e') {
															goto l88
														}
														position++
														if buffer[position] != rune('r') {
															goto l88
														}
														position++
														depth--
														add(rulePegText, position130)
													}
													{
														add(ruleAction16, position)
													}
													if !_rules[ruleWSX]() {
														goto l88
													}
													if !_rules[ruleValueCompare]() {
														goto l88
													}
													if !_rules[ruleWSX]() {
														goto l88
													}
													if !_rules[rulePublisherId]() {
														goto l88
													}
													{
														add(ruleAction17, position)
													}
													depth--
													add(rulePublisherCriteria, position129)
												}
												break
											default:
												{
													position133 := position
													depth++
													{
														position134 := position
														depth++
														if buffer[position] != rune('i') {
															goto l88
														}
														position++
														if buffer[position] != rune('d') {
															goto l88
														}
														position++
														depth--
														add(rulePegText, position134)
													}
													{
														add(ruleAction14, position)
													}
													if !_rules[ruleWSX]() {
														goto l88
													}
													if !_rules[ruleValueCompare]() {
														goto l88
													}
													if !_rules[ruleWSX]() {
														goto l88
													}
													{
														position136 := position
														depth++
														{
															position137 := position
															depth++
															{
																switch buffer[position] {
																case ':':
																	if buffer[position] != rune(':') {
																		goto l88
																	}
																	position++
																	break
																case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
																	if c := buffer[position]; c < rune('0') || c > rune('9') {
																		goto l88
																	}
																	position++
																	break
																case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
																	if c := buffer[position]; c < rune('A') || c > rune('Z') {
																		goto l88
																	}
																	position++
																	break
																default:
																	if c := buffer[position]; c < rune('a') || c > rune('z') {
																		goto l88
																	}
																	position++
																	break
																}
															}

														l138:
															{
																position139, tokenIndex139, depth139 := position, tokenIndex, depth
																{
																	switch buffer[position] {
																	case ':':
																		if buffer[position] != rune(':') {
																			goto l139
																		}
																		position++
																		break
																	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
																		if c := buffer[position]; c < rune('0') || c > rune('9') {
																			goto l139
																		}
																		position++
																		break
																	case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
																		if c := buffer[position]; c < rune('A') || c > rune('Z') {
																			goto l139
																		}
																		position++
																		break
																	default:
																		if c := buffer[position]; c < rune('a') || c > rune('z') {
																			goto l139
																		}
																		position++
																		break
																	}
																}

																goto l138
															l139:
																position, tokenIndex, depth = position139, tokenIndex139, depth139
															}
															depth--
															add(rulePegText, position137)
														}
														depth--
														add(ruleStatementId, position136)
													}
													{
														add(ruleAction15, position)
													}
													depth--
													add(ruleIdCriteria, position133)
												}
												break
											}
										}

										depth--
										add(ruleValueCriteria, position123)
									}
									{
										add(ruleAction11, position)
									}
									break
								}
							}

							depth--
							add(ruleSimpleCriteria, position92)
						}
						break
					}
				}

				depth--
				add(ruleCompoundCriteria, position89)
			}
			return true
		l88:
			position, tokenIndex, depth = position88, tokenIndex88, depth88
			return false
		},
		/* 17 SimpleCriteria <- <((&('w') (IndexCriteria Action13)) | (&('c' | 't') (RangeCriteria Action12)) | (&('i' | 'p' | 's') (ValueCriteria Action11)))> */
		nil,
		/* 18 ValueCriteria <- <((&('s') SourceCriteria) | (&('p') PublisherCriteria) | (&('i') IdCriteria))> */
		nil,
		/* 19 IdCriteria <- <(<('i' 'd')> Action14 WSX ValueCompare WSX StatementId Action15)> */
		nil,
		/* 20 PublisherCriteria <- <(<('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')> Action16 WSX ValueCompare WSX PublisherId Action17)> */
		nil,
		/* 21 SourceCriteria <- <(<('s' 'o' 'u' 'r' 'c' 'e')> Action18 WSX ValueCompare WSX PublisherId Action19)> */
		nil,
		/* 22 ValueCompare <- <(<ValueCompareOp> Action20)> */
		func() bool {
			position149, tokenIndex149, depth149 := position, tokenIndex, depth
			{
				position150 := position
				depth++
				{
					position151 := position
					depth++
					{
						position152 := position
						depth++
						{
							position153, tokenIndex153, depth153 := position, tokenIndex, depth
							if buffer[position] != rune('=') {
								goto l154
							}
							position++
							goto l153
						l154:
							position, tokenIndex, depth = position153, tokenIndex153, depth153
							if buffer[position] != rune('!') {
								goto l149
							}
							position++
							if buffer[position] != rune('=') {
								goto l149
							}
							position++
						}
					l153:
						depth--
						add(ruleValueCompareOp, position152)
					}
					depth--
					add(rulePegText, position151)
				}
				{
					add(ruleAction20, position)
				}
				depth--
				add(ruleValueCompare, position150)
			}
			return true
		l149:
			position, tokenIndex, depth = position149, tokenIndex149, depth149
			return false
		},
		/* 23 ValueCompareOp <- <('=' / ('!' '='))> */
		nil,
		/* 24 RangeCriteria <- <(RangeSelector WSX Comparison WSX UInt Action21)> */
		nil,
		/* 25 RangeSelector <- <(<RangeSelectorOp> Action22)> */
		nil,
		/* 26 RangeSelectorOp <- <(('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p') / ('c' 'o' 'u' 'n' 't' 'e' 'r'))> */
		nil,
		/* 27 Boolean <- <(<BooleanOp> Action23)> */
		nil,
		/* 28 BooleanOp <- <(('A' 'N' 'D') / ('O' 'R'))> */
		nil,
		/* 29 Comparison <- <(<ComparisonOp> Action24)> */
		nil,
		/* 30 ComparisonOp <- <(('<' '=') / ('>' '=') / ((&('>') '>') | (&('!') ('!' '=')) | (&('=') '=') | (&('<') '<')))> */
		nil,
		/* 31 IndexCriteria <- <WKICriteria> */
		nil,
		/* 32 WKICriteria <- <(<('w' 'k' 'i')> Action25 WSX '=' WSX WKI Action26)> */
		nil,
		/* 33 Order <- <('O' 'R' 'D' 'E' 'R' WS ('B' 'Y') WS OrderSpec Action27)> */
		nil,
		/* 34 OrderSpec <- <(OrderSelectorSpec (',' WSX OrderSelectorSpec)*)> */
		nil,
		/* 35 OrderSelectorSpec <- <(OrderSelector Action28 (WS OrderDir Action29)?)> */
		func() bool {
			position168, tokenIndex168, depth168 := position, tokenIndex, depth
			{
				position169 := position
				depth++
				{
					position170 := position
					depth++
					{
						position171 := position
						depth++
						{
							position172 := position
							depth++
							{
								switch buffer[position] {
								case 'c':
									if buffer[position] != rune('c') {
										goto l168
									}
									position++
									if buffer[position] != rune('o') {
										goto l168
									}
									position++
									if buffer[position] != rune('u') {
										goto l168
									}
									position++
									if buffer[position] != rune('n') {
										goto l168
									}
									position++
									if buffer[position] != rune('t') {
										goto l168
									}
									position++
									if buffer[position] != rune('e') {
										goto l168
									}
									position++
									if buffer[position] != rune('r') {
										goto l168
									}
									position++
									break
								case 't':
									if buffer[position] != rune('t') {
										goto l168
									}
									position++
									if buffer[position] != rune('i') {
										goto l168
									}
									position++
									if buffer[position] != rune('m') {
										goto l168
									}
									position++
									if buffer[position] != rune('e') {
										goto l168
									}
									position++
									if buffer[position] != rune('s') {
										goto l168
									}
									position++
									if buffer[position] != rune('t') {
										goto l168
									}
									position++
									if buffer[position] != rune('a') {
										goto l168
									}
									position++
									if buffer[position] != rune('m') {
										goto l168
									}
									position++
									if buffer[position] != rune('p') {
										goto l168
									}
									position++
									break
								case 's':
									if buffer[position] != rune('s') {
										goto l168
									}
									position++
									if buffer[position] != rune('o') {
										goto l168
									}
									position++
									if buffer[position] != rune('u') {
										goto l168
									}
									position++
									if buffer[position] != rune('r') {
										goto l168
									}
									position++
									if buffer[position] != rune('c') {
										goto l168
									}
									position++
									if buffer[position] != rune('e') {
										goto l168
									}
									position++
									break
								case 'p':
									if buffer[position] != rune('p') {
										goto l168
									}
									position++
									if buffer[position] != rune('u') {
										goto l168
									}
									position++
									if buffer[position] != rune('b') {
										goto l168
									}
									position++
									if buffer[position] != rune('l') {
										goto l168
									}
									position++
									if buffer[position] != rune('i') {
										goto l168
									}
									position++
									if buffer[position] != rune('s') {
										goto l168
									}
									position++
									if buffer[position] != rune('h') {
										goto l168
									}
									position++
									if buffer[position] != rune('e') {
										goto l168
									}
									position++
									if buffer[position] != rune('r') {
										goto l168
									}
									position++
									break
								case 'n':
									if buffer[position] != rune('n') {
										goto l168
									}
									position++
									if buffer[position] != rune('a') {
										goto l168
									}
									position++
									if buffer[position] != rune('m') {
										goto l168
									}
									position++
									if buffer[position] != rune('e') {
										goto l168
									}
									position++
									if buffer[position] != rune('s') {
										goto l168
									}
									position++
									if buffer[position] != rune('p') {
										goto l168
									}
									position++
									if buffer[position] != rune('a') {
										goto l168
									}
									position++
									if buffer[position] != rune('c') {
										goto l168
									}
									position++
									if buffer[position] != rune('e') {
										goto l168
									}
									position++
									break
								default:
									if buffer[position] != rune('i') {
										goto l168
									}
									position++
									if buffer[position] != rune('d') {
										goto l168
									}
									position++
									break
								}
							}

							depth--
							add(ruleOrderSelectorOp, position172)
						}
						depth--
						add(rulePegText, position171)
					}
					{
						add(ruleAction30, position)
					}
					depth--
					add(ruleOrderSelector, position170)
				}
				{
					add(ruleAction28, position)
				}
				{
					position176, tokenIndex176, depth176 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l176
					}
					{
						position178 := position
						depth++
						{
							position179 := position
							depth++
							{
								position180 := position
								depth++
								{
									position181, tokenIndex181, depth181 := position, tokenIndex, depth
									if buffer[position] != rune('A') {
										goto l182
									}
									position++
									if buffer[position] != rune('S') {
										goto l182
									}
									position++
									if buffer[position] != rune('C') {
										goto l182
									}
									position++
									goto l181
								l182:
									position, tokenIndex, depth = position181, tokenIndex181, depth181
									if buffer[position] != rune('D') {
										goto l176
									}
									position++
									if buffer[position] != rune('E') {
										goto l176
									}
									position++
									if buffer[position] != rune('S') {
										goto l176
									}
									position++
									if buffer[position] != rune('C') {
										goto l176
									}
									position++
								}
							l181:
								depth--
								add(ruleOrderDirOp, position180)
							}
							depth--
							add(rulePegText, position179)
						}
						{
							add(ruleAction31, position)
						}
						depth--
						add(ruleOrderDir, position178)
					}
					{
						add(ruleAction29, position)
					}
					goto l177
				l176:
					position, tokenIndex, depth = position176, tokenIndex176, depth176
				}
			l177:
				depth--
				add(ruleOrderSelectorSpec, position169)
			}
			return true
		l168:
			position, tokenIndex, depth = position168, tokenIndex168, depth168
			return false
		},
		/* 36 OrderSelector <- <(<OrderSelectorOp> Action30)> */
		nil,
		/* 37 OrderSelectorOp <- <((&('c') ('c' 'o' 'u' 'n' 't' 'e' 'r')) | (&('t') ('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p')) | (&('s') ('s' 'o' 'u' 'r' 'c' 'e')) | (&('p') ('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')) | (&('n') ('n' 'a' 'm' 'e' 's' 'p' 'a' 'c' 'e')) | (&('i') ('i' 'd')))> */
		nil,
		/* 38 OrderDir <- <(<OrderDirOp> Action31)> */
		nil,
		/* 39 OrderDirOp <- <(('A' 'S' 'C') / ('D' 'E' 'S' 'C'))> */
		nil,
		/* 40 Limit <- <('L' 'I' 'M' 'I' 'T' WS UInt Action32)> */
		nil,
		/* 41 StatementId <- <<((&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 42 PublisherId <- <<((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		func() bool {
			position191, tokenIndex191, depth191 := position, tokenIndex, depth
			{
				position192 := position
				depth++
				{
					position193 := position
					depth++
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l191
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l191
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l191
							}
							position++
							break
						}
					}

				l194:
					{
						position195, tokenIndex195, depth195 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l195
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l195
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l195
								}
								position++
								break
							}
						}

						goto l194
					l195:
						position, tokenIndex, depth = position195, tokenIndex195, depth195
					}
					depth--
					add(rulePegText, position193)
				}
				depth--
				add(rulePublisherId, position192)
			}
			return true
		l191:
			position, tokenIndex, depth = position191, tokenIndex191, depth191
			return false
		},
		/* 43 WKI <- <<((&('_') '_') | (&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('-') '-') | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 44 UInt <- <<[0-9]+>> */
		func() bool {
			position199, tokenIndex199, depth199 := position, tokenIndex, depth
			{
				position200 := position
				depth++
				{
					position201 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l199
					}
					position++
				l202:
					{
						position203, tokenIndex203, depth203 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l203
						}
						position++
						goto l202
					l203:
						position, tokenIndex, depth = position203, tokenIndex203, depth203
					}
					depth--
					add(rulePegText, position201)
				}
				depth--
				add(ruleUInt, position200)
			}
			return true
		l199:
			position, tokenIndex, depth = position199, tokenIndex199, depth199
			return false
		},
		/* 45 WS <- <WhiteSpace+> */
		func() bool {
			position204, tokenIndex204, depth204 := position, tokenIndex, depth
			{
				position205 := position
				depth++
				if !_rules[ruleWhiteSpace]() {
					goto l204
				}
			l206:
				{
					position207, tokenIndex207, depth207 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l207
					}
					goto l206
				l207:
					position, tokenIndex, depth = position207, tokenIndex207, depth207
				}
				depth--
				add(ruleWS, position205)
			}
			return true
		l204:
			position, tokenIndex, depth = position204, tokenIndex204, depth204
			return false
		},
		/* 46 WSX <- <WhiteSpace*> */
		func() bool {
			{
				position209 := position
				depth++
			l210:
				{
					position211, tokenIndex211, depth211 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l211
					}
					goto l210
				l211:
					position, tokenIndex, depth = position211, tokenIndex211, depth211
				}
				depth--
				add(ruleWSX, position209)
			}
			return true
		},
		/* 47 WhiteSpace <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') EOL))> */
		func() bool {
			position212, tokenIndex212, depth212 := position, tokenIndex, depth
			{
				position213 := position
				depth++
				{
					switch buffer[position] {
					case '\t':
						if buffer[position] != rune('\t') {
							goto l212
						}
						position++
						break
					case ' ':
						if buffer[position] != rune(' ') {
							goto l212
						}
						position++
						break
					default:
						{
							position215 := position
							depth++
							{
								position216, tokenIndex216, depth216 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l217
								}
								position++
								if buffer[position] != rune('\n') {
									goto l217
								}
								position++
								goto l216
							l217:
								position, tokenIndex, depth = position216, tokenIndex216, depth216
								if buffer[position] != rune('\n') {
									goto l218
								}
								position++
								goto l216
							l218:
								position, tokenIndex, depth = position216, tokenIndex216, depth216
								if buffer[position] != rune('\r') {
									goto l212
								}
								position++
							}
						l216:
							depth--
							add(ruleEOL, position215)
						}
						break
					}
				}

				depth--
				add(ruleWhiteSpace, position213)
			}
			return true
		l212:
			position, tokenIndex, depth = position212, tokenIndex212, depth212
			return false
		},
		/* 48 EOL <- <(('\r' '\n') / '\n' / '\r')> */
		nil,
		/* 49 EOF <- <!.> */
		func() bool {
			position220, tokenIndex220, depth220 := position, tokenIndex, depth
			{
				position221 := position
				depth++
				{
					position222, tokenIndex222, depth222 := position, tokenIndex, depth
					if !matchDot() {
						goto l222
					}
					goto l220
				l222:
					position, tokenIndex, depth = position222, tokenIndex222, depth222
				}
				depth--
				add(ruleEOF, position221)
			}
			return true
		l220:
			position, tokenIndex, depth = position220, tokenIndex220, depth220
			return false
		},
		/* 51 Action0 <- <{ p.setSelectOp() }> */
		nil,
		/* 52 Action1 <- <{ p.setDeleteOp() }> */
		nil,
		/* 53 Action2 <- <{ p.setSimpleSelector() }> */
		nil,
		/* 54 Action3 <- <{ p.setCompoundSelector() }> */
		nil,
		/* 55 Action4 <- <{ p.setFunctionSelector() }> */
		nil,
		nil,
		/* 57 Action5 <- <{ p.push(text) }> */
		nil,
		/* 58 Action6 <- <{ p.push(text) }> */
		nil,
		/* 59 Action7 <- <{ p.setNamespace(text) }> */
		nil,
		/* 60 Action8 <- <{ p.setCriteria() }> */
		nil,
		/* 61 Action9 <- <{ p.addCompoundCriteria() }> */
		nil,
		/* 62 Action10 <- <{ p.addNegatedCriteria() }> */
		nil,
		/* 63 Action11 <- <{ p.addValueCriteria() }> */
		nil,
		/* 64 Action12 <- <{ p.addRangeCriteria() }> */
		nil,
		/* 65 Action13 <- <{ p.addIndexCriteria() }> */
		nil,
		/* 66 Action14 <- <{ p.push(text) }> */
		nil,
		/* 67 Action15 <- <{ p.push(text) }> */
		nil,
		/* 68 Action16 <- <{ p.push(text) }> */
		nil,
		/* 69 Action17 <- <{ p.push(text) }> */
		nil,
		/* 70 Action18 <- <{ p.push(text) }> */
		nil,
		/* 71 Action19 <- <{ p.push(text) }> */
		nil,
		/* 72 Action20 <- <{ p.push(text) }> */
		nil,
		/* 73 Action21 <- <{ p.push(text) }> */
		nil,
		/* 74 Action22 <- <{ p.push(text) }> */
		nil,
		/* 75 Action23 <- <{ p.push(text) }> */
		nil,
		/* 76 Action24 <- <{ p.push(text) }> */
		nil,
		/* 77 Action25 <- <{ p.push(text) }> */
		nil,
		/* 78 Action26 <- <{ p.push(text) }> */
		nil,
		/* 79 Action27 <- <{ p.setOrder() }> */
		nil,
		/* 80 Action28 <- <{ p.addOrderSelector() }> */
		nil,
		/* 81 Action29 <- <{ p.setOrderDir() }> */
		nil,
		/* 82 Action30 <- <{ p.push(text) }> */
		nil,
		/* 83 Action31 <- <{ p.push(text) }> */
		nil,
		/* 84 Action32 <- <{ p.setLimit(text) }> */
		nil,
	}
	p.rules = _rules
}
