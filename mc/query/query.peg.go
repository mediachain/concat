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
	ruleAction0
	ruleAction1
	ruleAction2
	rulePegText
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
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19

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
	"Action0",
	"Action1",
	"Action2",
	"PegText",
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
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",

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
	rules  [56]func() bool
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
			p.setSimpleSelector()
		case ruleAction1:
			p.setCompoundSelector()
		case ruleAction2:
			p.setFunctionSelector()
		case ruleAction3:
			p.push(text)
		case ruleAction4:
			p.push(text)
		case ruleAction5:
			p.setNamespace(text)
		case ruleAction6:
			p.setCriteria()
		case ruleAction7:
			p.addCompoundCriteria()
		case ruleAction8:
			p.addValueCriteria()
		case ruleAction9:
			p.addTimeCriteria()
		case ruleAction10:
			p.push(text)
		case ruleAction11:
			p.push(text)
		case ruleAction12:
			p.push(text)
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
									{
										add(ruleAction4, position)
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
							{
								add(ruleAction2, position)
							}
							break
						case '(':
							{
								position10 := position
								depth++
								if buffer[position] != rune('(') {
									goto l0
								}
								position++
								if !_rules[ruleSimpleSelector]() {
									goto l0
								}
							l11:
								{
									position12, tokenIndex12, depth12 := position, tokenIndex, depth
									if buffer[position] != rune(',') {
										goto l12
									}
									position++
									if !_rules[ruleWSX]() {
										goto l12
									}
									if !_rules[ruleSimpleSelector]() {
										goto l12
									}
									goto l11
								l12:
									position, tokenIndex, depth = position12, tokenIndex12, depth12
								}
								if buffer[position] != rune(')') {
									goto l0
								}
								position++
								depth--
								add(ruleCompoundSelector, position10)
							}
							{
								add(ruleAction1, position)
							}
							break
						default:
							if !_rules[ruleSimpleSelector]() {
								goto l0
							}
							{
								add(ruleAction0, position)
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
					position15 := position
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
						position16 := position
						depth++
						{
							position17, tokenIndex17, depth17 := position, tokenIndex, depth
							{
								position19 := position
								depth++
								if !_rules[ruleNamespacePart]() {
									goto l18
								}
							l20:
								{
									position21, tokenIndex21, depth21 := position, tokenIndex, depth
									if buffer[position] != rune('.') {
										goto l21
									}
									position++
									if !_rules[ruleNamespacePart]() {
										goto l21
									}
									goto l20
								l21:
									position, tokenIndex, depth = position21, tokenIndex21, depth21
								}
								{
									position22, tokenIndex22, depth22 := position, tokenIndex, depth
									if buffer[position] != rune('.') {
										goto l22
									}
									position++
									if !_rules[ruleWildcard]() {
										goto l22
									}
									goto l23
								l22:
									position, tokenIndex, depth = position22, tokenIndex22, depth22
								}
							l23:
								depth--
								add(rulePegText, position19)
							}
							goto l17
						l18:
							position, tokenIndex, depth = position17, tokenIndex17, depth17
							{
								position24 := position
								depth++
								if !_rules[ruleWildcard]() {
									goto l0
								}
								depth--
								add(rulePegText, position24)
							}
						}
					l17:
						depth--
						add(ruleNamespace, position16)
					}
					{
						add(ruleAction5, position)
					}
					depth--
					add(ruleSource, position15)
				}
				{
					position26, tokenIndex26, depth26 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l26
					}
					{
						position28 := position
						depth++
						if buffer[position] != rune('W') {
							goto l26
						}
						position++
						if buffer[position] != rune('H') {
							goto l26
						}
						position++
						if buffer[position] != rune('E') {
							goto l26
						}
						position++
						if buffer[position] != rune('R') {
							goto l26
						}
						position++
						if buffer[position] != rune('E') {
							goto l26
						}
						position++
						if !_rules[ruleWS]() {
							goto l26
						}
						if !_rules[ruleMultiCriteria]() {
							goto l26
						}
						{
							add(ruleAction6, position)
						}
						depth--
						add(ruleCriteria, position28)
					}
					goto l27
				l26:
					position, tokenIndex, depth = position26, tokenIndex26, depth26
				}
			l27:
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
							add(ruleAction19, position)
						}
						depth--
						add(ruleLimit, position32)
					}
					goto l31
				l30:
					position, tokenIndex, depth = position30, tokenIndex30, depth30
				}
			l31:
				if !_rules[ruleWSX]() {
					goto l0
				}
				{
					position34 := position
					depth++
					{
						position35, tokenIndex35, depth35 := position, tokenIndex, depth
						if !matchDot() {
							goto l35
						}
						goto l0
					l35:
						position, tokenIndex, depth = position35, tokenIndex35, depth35
					}
					depth--
					add(ruleEOF, position34)
				}
				depth--
				add(ruleGrammar, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 Selector <- <((&('C') (FunctionSelector Action2)) | (&('(') (CompoundSelector Action1)) | (&('*' | 'b' | 'i' | 'n' | 'p' | 's' | 't') (SimpleSelector Action0)))> */
		nil,
		/* 2 SimpleSelector <- <(<SimpleSelectorOp> Action3)> */
		func() bool {
			position37, tokenIndex37, depth37 := position, tokenIndex, depth
			{
				position38 := position
				depth++
				{
					position39 := position
					depth++
					{
						position40 := position
						depth++
						{
							switch buffer[position] {
							case 't':
								if buffer[position] != rune('t') {
									goto l37
								}
								position++
								if buffer[position] != rune('i') {
									goto l37
								}
								position++
								if buffer[position] != rune('m') {
									goto l37
								}
								position++
								if buffer[position] != rune('e') {
									goto l37
								}
								position++
								if buffer[position] != rune('s') {
									goto l37
								}
								position++
								if buffer[position] != rune('t') {
									goto l37
								}
								position++
								if buffer[position] != rune('a') {
									goto l37
								}
								position++
								if buffer[position] != rune('m') {
									goto l37
								}
								position++
								if buffer[position] != rune('p') {
									goto l37
								}
								position++
								break
							case 's':
								if buffer[position] != rune('s') {
									goto l37
								}
								position++
								if buffer[position] != rune('o') {
									goto l37
								}
								position++
								if buffer[position] != rune('u') {
									goto l37
								}
								position++
								if buffer[position] != rune('r') {
									goto l37
								}
								position++
								if buffer[position] != rune('c') {
									goto l37
								}
								position++
								if buffer[position] != rune('e') {
									goto l37
								}
								position++
								break
							case 'n':
								if buffer[position] != rune('n') {
									goto l37
								}
								position++
								if buffer[position] != rune('a') {
									goto l37
								}
								position++
								if buffer[position] != rune('m') {
									goto l37
								}
								position++
								if buffer[position] != rune('e') {
									goto l37
								}
								position++
								if buffer[position] != rune('s') {
									goto l37
								}
								position++
								if buffer[position] != rune('p') {
									goto l37
								}
								position++
								if buffer[position] != rune('a') {
									goto l37
								}
								position++
								if buffer[position] != rune('c') {
									goto l37
								}
								position++
								if buffer[position] != rune('e') {
									goto l37
								}
								position++
								break
							case 'p':
								if buffer[position] != rune('p') {
									goto l37
								}
								position++
								if buffer[position] != rune('u') {
									goto l37
								}
								position++
								if buffer[position] != rune('b') {
									goto l37
								}
								position++
								if buffer[position] != rune('l') {
									goto l37
								}
								position++
								if buffer[position] != rune('i') {
									goto l37
								}
								position++
								if buffer[position] != rune('s') {
									goto l37
								}
								position++
								if buffer[position] != rune('h') {
									goto l37
								}
								position++
								if buffer[position] != rune('e') {
									goto l37
								}
								position++
								if buffer[position] != rune('r') {
									goto l37
								}
								position++
								break
							case 'i':
								if buffer[position] != rune('i') {
									goto l37
								}
								position++
								if buffer[position] != rune('d') {
									goto l37
								}
								position++
								break
							case 'b':
								if buffer[position] != rune('b') {
									goto l37
								}
								position++
								if buffer[position] != rune('o') {
									goto l37
								}
								position++
								if buffer[position] != rune('d') {
									goto l37
								}
								position++
								if buffer[position] != rune('y') {
									goto l37
								}
								position++
								break
							default:
								if buffer[position] != rune('*') {
									goto l37
								}
								position++
								break
							}
						}

						depth--
						add(ruleSimpleSelectorOp, position40)
					}
					depth--
					add(rulePegText, position39)
				}
				{
					add(ruleAction3, position)
				}
				depth--
				add(ruleSimpleSelector, position38)
			}
			return true
		l37:
			position, tokenIndex, depth = position37, tokenIndex37, depth37
			return false
		},
		/* 3 SimpleSelectorOp <- <((&('t') ('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p')) | (&('s') ('s' 'o' 'u' 'r' 'c' 'e')) | (&('n') ('n' 'a' 'm' 'e' 's' 'p' 'a' 'c' 'e')) | (&('p') ('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')) | (&('i') ('i' 'd')) | (&('b') ('b' 'o' 'd' 'y')) | (&('*') '*'))> */
		nil,
		/* 4 CompoundSelector <- <('(' SimpleSelector (',' WSX SimpleSelector)* ')')> */
		nil,
		/* 5 FunctionSelector <- <(Function '(' SimpleSelector ')')> */
		nil,
		/* 6 Function <- <(<FunctionOp> Action4)> */
		nil,
		/* 7 FunctionOp <- <('C' 'O' 'U' 'N' 'T')> */
		nil,
		/* 8 Source <- <('F' 'R' 'O' 'M' WS Namespace Action5)> */
		nil,
		/* 9 Namespace <- <(<(NamespacePart ('.' NamespacePart)* ('.' Wildcard)?)> / <Wildcard>)> */
		nil,
		/* 10 NamespacePart <- <((&('-') '-') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position50, tokenIndex50, depth50 := position, tokenIndex, depth
			{
				position51 := position
				depth++
				{
					switch buffer[position] {
					case '-':
						if buffer[position] != rune('-') {
							goto l50
						}
						position++
						break
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l50
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l50
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l50
						}
						position++
						break
					}
				}

			l52:
				{
					position53, tokenIndex53, depth53 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '-':
							if buffer[position] != rune('-') {
								goto l53
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l53
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l53
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l53
							}
							position++
							break
						}
					}

					goto l52
				l53:
					position, tokenIndex, depth = position53, tokenIndex53, depth53
				}
				depth--
				add(ruleNamespacePart, position51)
			}
			return true
		l50:
			position, tokenIndex, depth = position50, tokenIndex50, depth50
			return false
		},
		/* 11 Wildcard <- <'*'> */
		func() bool {
			position56, tokenIndex56, depth56 := position, tokenIndex, depth
			{
				position57 := position
				depth++
				if buffer[position] != rune('*') {
					goto l56
				}
				position++
				depth--
				add(ruleWildcard, position57)
			}
			return true
		l56:
			position, tokenIndex, depth = position56, tokenIndex56, depth56
			return false
		},
		/* 12 Criteria <- <('W' 'H' 'E' 'R' 'E' WS MultiCriteria Action6)> */
		nil,
		/* 13 MultiCriteria <- <(CompoundCriteria (WS Boolean WS CompoundCriteria Action7)*)> */
		func() bool {
			position59, tokenIndex59, depth59 := position, tokenIndex, depth
			{
				position60 := position
				depth++
				if !_rules[ruleCompoundCriteria]() {
					goto l59
				}
			l61:
				{
					position62, tokenIndex62, depth62 := position, tokenIndex, depth
					if !_rules[ruleWS]() {
						goto l62
					}
					{
						position63 := position
						depth++
						{
							position64 := position
							depth++
							{
								position65 := position
								depth++
								{
									position66, tokenIndex66, depth66 := position, tokenIndex, depth
									if buffer[position] != rune('A') {
										goto l67
									}
									position++
									if buffer[position] != rune('N') {
										goto l67
									}
									position++
									if buffer[position] != rune('D') {
										goto l67
									}
									position++
									goto l66
								l67:
									position, tokenIndex, depth = position66, tokenIndex66, depth66
									if buffer[position] != rune('O') {
										goto l62
									}
									position++
									if buffer[position] != rune('R') {
										goto l62
									}
									position++
								}
							l66:
								depth--
								add(ruleBooleanOp, position65)
							}
							depth--
							add(rulePegText, position64)
						}
						{
							add(ruleAction17, position)
						}
						depth--
						add(ruleBoolean, position63)
					}
					if !_rules[ruleWS]() {
						goto l62
					}
					if !_rules[ruleCompoundCriteria]() {
						goto l62
					}
					{
						add(ruleAction7, position)
					}
					goto l61
				l62:
					position, tokenIndex, depth = position62, tokenIndex62, depth62
				}
				depth--
				add(ruleMultiCriteria, position60)
			}
			return true
		l59:
			position, tokenIndex, depth = position59, tokenIndex59, depth59
			return false
		},
		/* 14 CompoundCriteria <- <(SimpleCriteria / ('(' MultiCriteria ')'))> */
		func() bool {
			position70, tokenIndex70, depth70 := position, tokenIndex, depth
			{
				position71 := position
				depth++
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					{
						position74 := position
						depth++
						{
							position75, tokenIndex75, depth75 := position, tokenIndex, depth
							{
								position77 := position
								depth++
								{
									switch buffer[position] {
									case 's':
										{
											position79 := position
											depth++
											{
												position80 := position
												depth++
												if buffer[position] != rune('s') {
													goto l76
												}
												position++
												if buffer[position] != rune('o') {
													goto l76
												}
												position++
												if buffer[position] != rune('u') {
													goto l76
												}
												position++
												if buffer[position] != rune('r') {
													goto l76
												}
												position++
												if buffer[position] != rune('c') {
													goto l76
												}
												position++
												if buffer[position] != rune('e') {
													goto l76
												}
												position++
												depth--
												add(rulePegText, position80)
											}
											{
												add(ruleAction14, position)
											}
											if !_rules[ruleWSX]() {
												goto l76
											}
											if buffer[position] != rune('=') {
												goto l76
											}
											position++
											if !_rules[ruleWSX]() {
												goto l76
											}
											if !_rules[rulePublisherId]() {
												goto l76
											}
											{
												add(ruleAction15, position)
											}
											depth--
											add(ruleSourceCriteria, position79)
										}
										break
									case 'p':
										{
											position83 := position
											depth++
											{
												position84 := position
												depth++
												if buffer[position] != rune('p') {
													goto l76
												}
												position++
												if buffer[position] != rune('u') {
													goto l76
												}
												position++
												if buffer[position] != rune('b') {
													goto l76
												}
												position++
												if buffer[position] != rune('l') {
													goto l76
												}
												position++
												if buffer[position] != rune('i') {
													goto l76
												}
												position++
												if buffer[position] != rune('s') {
													goto l76
												}
												position++
												if buffer[position] != rune('h') {
													goto l76
												}
												position++
												if buffer[position] != rune('e') {
													goto l76
												}
												position++
												if buffer[position] != rune('r') {
													goto l76
												}
												position++
												depth--
												add(rulePegText, position84)
											}
											{
												add(ruleAction12, position)
											}
											if !_rules[ruleWSX]() {
												goto l76
											}
											if buffer[position] != rune('=') {
												goto l76
											}
											position++
											if !_rules[ruleWSX]() {
												goto l76
											}
											if !_rules[rulePublisherId]() {
												goto l76
											}
											{
												add(ruleAction13, position)
											}
											depth--
											add(rulePublisherCriteria, position83)
										}
										break
									default:
										{
											position87 := position
											depth++
											{
												position88 := position
												depth++
												if buffer[position] != rune('i') {
													goto l76
												}
												position++
												if buffer[position] != rune('d') {
													goto l76
												}
												position++
												depth--
												add(rulePegText, position88)
											}
											{
												add(ruleAction10, position)
											}
											if !_rules[ruleWSX]() {
												goto l76
											}
											if buffer[position] != rune('=') {
												goto l76
											}
											position++
											if !_rules[ruleWSX]() {
												goto l76
											}
											{
												position90 := position
												depth++
												{
													position91 := position
													depth++
													{
														switch buffer[position] {
														case ':':
															if buffer[position] != rune(':') {
																goto l76
															}
															position++
															break
														case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
															if c := buffer[position]; c < rune('0') || c > rune('9') {
																goto l76
															}
															position++
															break
														case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
															if c := buffer[position]; c < rune('A') || c > rune('Z') {
																goto l76
															}
															position++
															break
														default:
															if c := buffer[position]; c < rune('a') || c > rune('z') {
																goto l76
															}
															position++
															break
														}
													}

												l92:
													{
														position93, tokenIndex93, depth93 := position, tokenIndex, depth
														{
															switch buffer[position] {
															case ':':
																if buffer[position] != rune(':') {
																	goto l93
																}
																position++
																break
															case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
																if c := buffer[position]; c < rune('0') || c > rune('9') {
																	goto l93
																}
																position++
																break
															case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
																if c := buffer[position]; c < rune('A') || c > rune('Z') {
																	goto l93
																}
																position++
																break
															default:
																if c := buffer[position]; c < rune('a') || c > rune('z') {
																	goto l93
																}
																position++
																break
															}
														}

														goto l92
													l93:
														position, tokenIndex, depth = position93, tokenIndex93, depth93
													}
													depth--
													add(rulePegText, position91)
												}
												depth--
												add(ruleStatementId, position90)
											}
											{
												add(ruleAction11, position)
											}
											depth--
											add(ruleIdCriteria, position87)
										}
										break
									}
								}

								depth--
								add(ruleValueCriteria, position77)
							}
							{
								add(ruleAction8, position)
							}
							goto l75
						l76:
							position, tokenIndex, depth = position75, tokenIndex75, depth75
							{
								position98 := position
								depth++
								if buffer[position] != rune('t') {
									goto l73
								}
								position++
								if buffer[position] != rune('i') {
									goto l73
								}
								position++
								if buffer[position] != rune('m') {
									goto l73
								}
								position++
								if buffer[position] != rune('e') {
									goto l73
								}
								position++
								if buffer[position] != rune('s') {
									goto l73
								}
								position++
								if buffer[position] != rune('t') {
									goto l73
								}
								position++
								if buffer[position] != rune('a') {
									goto l73
								}
								position++
								if buffer[position] != rune('m') {
									goto l73
								}
								position++
								if buffer[position] != rune('p') {
									goto l73
								}
								position++
								if !_rules[ruleWSX]() {
									goto l73
								}
								{
									position99 := position
									depth++
									{
										position100 := position
										depth++
										{
											position101 := position
											depth++
											{
												position102, tokenIndex102, depth102 := position, tokenIndex, depth
												if buffer[position] != rune('<') {
													goto l103
												}
												position++
												if buffer[position] != rune('=') {
													goto l103
												}
												position++
												goto l102
											l103:
												position, tokenIndex, depth = position102, tokenIndex102, depth102
												if buffer[position] != rune('>') {
													goto l104
												}
												position++
												if buffer[position] != rune('=') {
													goto l104
												}
												position++
												goto l102
											l104:
												position, tokenIndex, depth = position102, tokenIndex102, depth102
												{
													switch buffer[position] {
													case '>':
														if buffer[position] != rune('>') {
															goto l73
														}
														position++
														break
													case '!':
														if buffer[position] != rune('!') {
															goto l73
														}
														position++
														if buffer[position] != rune('=') {
															goto l73
														}
														position++
														break
													case '=':
														if buffer[position] != rune('=') {
															goto l73
														}
														position++
														break
													default:
														if buffer[position] != rune('<') {
															goto l73
														}
														position++
														break
													}
												}

											}
										l102:
											depth--
											add(ruleComparisonOp, position101)
										}
										depth--
										add(rulePegText, position100)
									}
									{
										add(ruleAction18, position)
									}
									depth--
									add(ruleComparison, position99)
								}
								if !_rules[ruleWSX]() {
									goto l73
								}
								if !_rules[ruleUInt]() {
									goto l73
								}
								{
									add(ruleAction16, position)
								}
								depth--
								add(ruleTimeCriteria, position98)
							}
							{
								add(ruleAction9, position)
							}
						}
					l75:
						depth--
						add(ruleSimpleCriteria, position74)
					}
					goto l72
				l73:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
					if buffer[position] != rune('(') {
						goto l70
					}
					position++
					if !_rules[ruleMultiCriteria]() {
						goto l70
					}
					if buffer[position] != rune(')') {
						goto l70
					}
					position++
				}
			l72:
				depth--
				add(ruleCompoundCriteria, position71)
			}
			return true
		l70:
			position, tokenIndex, depth = position70, tokenIndex70, depth70
			return false
		},
		/* 15 SimpleCriteria <- <((ValueCriteria Action8) / (TimeCriteria Action9))> */
		nil,
		/* 16 ValueCriteria <- <((&('s') SourceCriteria) | (&('p') PublisherCriteria) | (&('i') IdCriteria))> */
		nil,
		/* 17 IdCriteria <- <(<('i' 'd')> Action10 WSX '=' WSX StatementId Action11)> */
		nil,
		/* 18 PublisherCriteria <- <(<('p' 'u' 'b' 'l' 'i' 's' 'h' 'e' 'r')> Action12 WSX '=' WSX PublisherId Action13)> */
		nil,
		/* 19 SourceCriteria <- <(<('s' 'o' 'u' 'r' 'c' 'e')> Action14 WSX '=' WSX PublisherId Action15)> */
		nil,
		/* 20 TimeCriteria <- <('t' 'i' 'm' 'e' 's' 't' 'a' 'm' 'p' WSX Comparison WSX UInt Action16)> */
		nil,
		/* 21 Boolean <- <(<BooleanOp> Action17)> */
		nil,
		/* 22 BooleanOp <- <(('A' 'N' 'D') / ('O' 'R'))> */
		nil,
		/* 23 Comparison <- <(<ComparisonOp> Action18)> */
		nil,
		/* 24 ComparisonOp <- <(('<' '=') / ('>' '=') / ((&('>') '>') | (&('!') ('!' '=')) | (&('=') '=') | (&('<') '<')))> */
		nil,
		/* 25 Limit <- <('L' 'I' 'M' 'I' 'T' WS UInt Action19)> */
		nil,
		/* 26 StatementId <- <<((&(':') ':') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		nil,
		/* 27 PublisherId <- <<((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+>> */
		func() bool {
			position121, tokenIndex121, depth121 := position, tokenIndex, depth
			{
				position122 := position
				depth++
				{
					position123 := position
					depth++
					{
						switch buffer[position] {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l121
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l121
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l121
							}
							position++
							break
						}
					}

				l124:
					{
						position125, tokenIndex125, depth125 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l125
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l125
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l125
								}
								position++
								break
							}
						}

						goto l124
					l125:
						position, tokenIndex, depth = position125, tokenIndex125, depth125
					}
					depth--
					add(rulePegText, position123)
				}
				depth--
				add(rulePublisherId, position122)
			}
			return true
		l121:
			position, tokenIndex, depth = position121, tokenIndex121, depth121
			return false
		},
		/* 28 UInt <- <<[0-9]+>> */
		func() bool {
			position128, tokenIndex128, depth128 := position, tokenIndex, depth
			{
				position129 := position
				depth++
				{
					position130 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l128
					}
					position++
				l131:
					{
						position132, tokenIndex132, depth132 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l132
						}
						position++
						goto l131
					l132:
						position, tokenIndex, depth = position132, tokenIndex132, depth132
					}
					depth--
					add(rulePegText, position130)
				}
				depth--
				add(ruleUInt, position129)
			}
			return true
		l128:
			position, tokenIndex, depth = position128, tokenIndex128, depth128
			return false
		},
		/* 29 WS <- <WhiteSpace+> */
		func() bool {
			position133, tokenIndex133, depth133 := position, tokenIndex, depth
			{
				position134 := position
				depth++
				if !_rules[ruleWhiteSpace]() {
					goto l133
				}
			l135:
				{
					position136, tokenIndex136, depth136 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l136
					}
					goto l135
				l136:
					position, tokenIndex, depth = position136, tokenIndex136, depth136
				}
				depth--
				add(ruleWS, position134)
			}
			return true
		l133:
			position, tokenIndex, depth = position133, tokenIndex133, depth133
			return false
		},
		/* 30 WSX <- <WhiteSpace*> */
		func() bool {
			{
				position138 := position
				depth++
			l139:
				{
					position140, tokenIndex140, depth140 := position, tokenIndex, depth
					if !_rules[ruleWhiteSpace]() {
						goto l140
					}
					goto l139
				l140:
					position, tokenIndex, depth = position140, tokenIndex140, depth140
				}
				depth--
				add(ruleWSX, position138)
			}
			return true
		},
		/* 31 WhiteSpace <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') EOL))> */
		func() bool {
			position141, tokenIndex141, depth141 := position, tokenIndex, depth
			{
				position142 := position
				depth++
				{
					switch buffer[position] {
					case '\t':
						if buffer[position] != rune('\t') {
							goto l141
						}
						position++
						break
					case ' ':
						if buffer[position] != rune(' ') {
							goto l141
						}
						position++
						break
					default:
						{
							position144 := position
							depth++
							{
								position145, tokenIndex145, depth145 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l146
								}
								position++
								if buffer[position] != rune('\n') {
									goto l146
								}
								position++
								goto l145
							l146:
								position, tokenIndex, depth = position145, tokenIndex145, depth145
								if buffer[position] != rune('\n') {
									goto l147
								}
								position++
								goto l145
							l147:
								position, tokenIndex, depth = position145, tokenIndex145, depth145
								if buffer[position] != rune('\r') {
									goto l141
								}
								position++
							}
						l145:
							depth--
							add(ruleEOL, position144)
						}
						break
					}
				}

				depth--
				add(ruleWhiteSpace, position142)
			}
			return true
		l141:
			position, tokenIndex, depth = position141, tokenIndex141, depth141
			return false
		},
		/* 32 EOL <- <(('\r' '\n') / '\n' / '\r')> */
		nil,
		/* 33 EOF <- <!.> */
		nil,
		/* 35 Action0 <- <{ p.setSimpleSelector() }> */
		nil,
		/* 36 Action1 <- <{ p.setCompoundSelector() }> */
		nil,
		/* 37 Action2 <- <{ p.setFunctionSelector() }> */
		nil,
		nil,
		/* 39 Action3 <- <{ p.push(text) }> */
		nil,
		/* 40 Action4 <- <{ p.push(text) }> */
		nil,
		/* 41 Action5 <- <{ p.setNamespace(text) }> */
		nil,
		/* 42 Action6 <- <{ p.setCriteria() }> */
		nil,
		/* 43 Action7 <- <{ p.addCompoundCriteria() }> */
		nil,
		/* 44 Action8 <- <{ p.addValueCriteria() }> */
		nil,
		/* 45 Action9 <- <{ p.addTimeCriteria() }> */
		nil,
		/* 46 Action10 <- <{ p.push(text) }> */
		nil,
		/* 47 Action11 <- <{ p.push(text) }> */
		nil,
		/* 48 Action12 <- <{ p.push(text) }> */
		nil,
		/* 49 Action13 <- <{ p.push(text) }> */
		nil,
		/* 50 Action14 <- <{ p.push(text) }> */
		nil,
		/* 51 Action15 <- <{ p.push(text) }> */
		nil,
		/* 52 Action16 <- <{ p.push(text) }> */
		nil,
		/* 53 Action17 <- <{ p.push(text) }> */
		nil,
		/* 54 Action18 <- <{ p.push(text) }> */
		nil,
		/* 55 Action19 <- <{ p.setLimit(text) }> */
		nil,
	}
	p.rules = _rules
}
