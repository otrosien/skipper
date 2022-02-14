// Code generated by goyacc -o parser.go -p eskip parser.y. DO NOT EDIT.

//line parser.y:16
//lint:file-ignore ST1016 This is a generated file.
//lint:file-ignore SA4006 This is a generated file.

package eskip

import __yyfmt__ "fmt"

//line parser.y:19

import "strconv"

// conversion error ignored, tokenizer expression already checked format
func convertNumber(s string) float64 {
	n, _ := strconv.ParseFloat(s, 64)
	return n
}

//line parser.y:31
type eskipSymType struct {
	yys         int
	token       string
	route       *parsedRoute
	routes      []*parsedRoute
	matchers    []*matcher
	matcher     *matcher
	filter      *Filter
	filters     []*Filter
	args        []interface{}
	arg         interface{}
	backend     string
	shunt       bool
	loopback    bool
	dynamic     bool
	lbBackend   bool
	numval      float64
	stringval   string
	regexpval   string
	stringvals  []string
	lbAlgorithm string
	lbEndpoints []string
}

const and = 57346
const any = 57347
const arrow = 57348
const closeparen = 57349
const colon = 57350
const comma = 57351
const number = 57352
const openparen = 57353
const regexpliteral = 57354
const semicolon = 57355
const shunt = 57356
const loopback = 57357
const dynamic = 57358
const stringliteral = 57359
const symbol = 57360
const openarrow = 57361
const closearrow = 57362

var eskipToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"and",
	"any",
	"arrow",
	"closeparen",
	"colon",
	"comma",
	"number",
	"openparen",
	"regexpliteral",
	"semicolon",
	"shunt",
	"loopback",
	"dynamic",
	"stringliteral",
	"symbol",
	"openarrow",
	"closearrow",
}

var eskipStatenames = [...]string{}

const eskipEofCode = 1
const eskipErrCode = 2
const eskipInitialStackSize = 16

//line parser.y:287

//line yacctab:1
var eskipExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const eskipPrivate = 57344

const eskipLast = 62

var eskipAct = [...]int{
	34, 40, 32, 31, 24, 17, 20, 21, 22, 25,
	27, 26, 19, 48, 36, 9, 37, 25, 41, 9,
	16, 25, 25, 3, 10, 7, 14, 42, 29, 43,
	4, 55, 8, 45, 44, 49, 45, 30, 28, 19,
	50, 15, 13, 47, 46, 38, 23, 51, 52, 39,
	53, 42, 54, 12, 35, 11, 33, 18, 5, 6,
	2, 1,
}

var eskipPact = [...]int{
	14, -1000, 11, -1000, -1000, 49, 34, -1000, 15, -1000,
	2, -8, 10, 10, 4, -1000, -1000, -1000, 39, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, 0, 18, -1000, 15,
	-1000, 27, -1000, -1000, -1000, -1000, -1000, -1000, -8, -7,
	26, 31, -1000, 4, -1000, 4, -1000, -1000, -1000, 5,
	5, 24, -1000, -1000, 26, -1000,
}

var eskipPgo = [...]int{
	0, 61, 60, 23, 30, 59, 58, 5, 57, 25,
	3, 4, 2, 56, 0, 54, 1, 49, 46,
}

var eskipR1 = [...]int{
	0, 1, 1, 2, 2, 2, 2, 4, 5, 3,
	3, 6, 6, 9, 9, 8, 8, 11, 10, 10,
	10, 12, 12, 12, 16, 16, 17, 17, 18, 7,
	7, 7, 7, 7, 13, 14, 15,
}

var eskipR2 = [...]int{
	0, 1, 1, 0, 1, 3, 2, 3, 1, 3,
	5, 1, 3, 1, 4, 1, 3, 4, 0, 1,
	3, 1, 1, 1, 1, 3, 1, 3, 3, 1,
	1, 1, 1, 1, 1, 1, 1,
}

var eskipChk = [...]int{
	-1000, -1, -2, -3, -4, -6, -5, -9, 18, 5,
	13, 6, 4, 8, 11, -4, 18, -7, -8, -14,
	14, 15, 16, -18, -11, 17, 19, 18, -9, 18,
	-3, -10, -12, -13, -14, -15, 10, 12, 6, -17,
	-16, 18, -14, 11, 7, 9, -7, -11, 20, 9,
	9, -10, -12, -14, -16, 7,
}

var eskipDef = [...]int{
	3, -2, 1, 2, 4, 0, 0, 11, 8, 13,
	6, 0, 0, 0, 18, 5, 8, 9, 0, 29,
	30, 31, 32, 33, 15, 35, 0, 0, 12, 0,
	7, 0, 19, 21, 22, 23, 34, 36, 0, 0,
	26, 0, 24, 18, 14, 0, 10, 16, 28, 0,
	0, 0, 20, 25, 27, 17,
}

var eskipTok1 = [...]int{
	1,
}

var eskipTok2 = [...]int{
	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20,
}

var eskipTok3 = [...]int{
	0,
}

var eskipErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	eskipDebug        = 0
	eskipErrorVerbose = false
)

type eskipLexer interface {
	Lex(lval *eskipSymType) int
	Error(s string)
}

type eskipParser interface {
	Parse(eskipLexer) int
	Lookahead() int
}

type eskipParserImpl struct {
	lval  eskipSymType
	stack [eskipInitialStackSize]eskipSymType
	char  int
}

func (p *eskipParserImpl) Lookahead() int {
	return p.char
}

func eskipNewParser() eskipParser {
	return &eskipParserImpl{}
}

const eskipFlag = -1000

func eskipTokname(c int) string {
	if c >= 1 && c-1 < len(eskipToknames) {
		if eskipToknames[c-1] != "" {
			return eskipToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func eskipStatname(s int) string {
	if s >= 0 && s < len(eskipStatenames) {
		if eskipStatenames[s] != "" {
			return eskipStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func eskipErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !eskipErrorVerbose {
		return "syntax error"
	}

	for _, e := range eskipErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + eskipTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := eskipPact[state]
	for tok := TOKSTART; tok-1 < len(eskipToknames); tok++ {
		if n := base + tok; n >= 0 && n < eskipLast && eskipChk[eskipAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if eskipDef[state] == -2 {
		i := 0
		for eskipExca[i] != -1 || eskipExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; eskipExca[i] >= 0; i += 2 {
			tok := eskipExca[i]
			if tok < TOKSTART || eskipExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if eskipExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += eskipTokname(tok)
	}
	return res
}

func eskiplex1(lex eskipLexer, lval *eskipSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = eskipTok1[0]
		goto out
	}
	if char < len(eskipTok1) {
		token = eskipTok1[char]
		goto out
	}
	if char >= eskipPrivate {
		if char < eskipPrivate+len(eskipTok2) {
			token = eskipTok2[char-eskipPrivate]
			goto out
		}
	}
	for i := 0; i < len(eskipTok3); i += 2 {
		token = eskipTok3[i+0]
		if token == char {
			token = eskipTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = eskipTok2[1] /* unknown char */
	}
	if eskipDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", eskipTokname(token), uint(char))
	}
	return char, token
}

func eskipParse(eskiplex eskipLexer) int {
	return eskipNewParser().Parse(eskiplex)
}

func (eskiprcvr *eskipParserImpl) Parse(eskiplex eskipLexer) int {
	var eskipn int
	var eskipVAL eskipSymType
	var eskipDollar []eskipSymType
	_ = eskipDollar // silence set and not used
	eskipS := eskiprcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	eskipstate := 0
	eskiprcvr.char = -1
	eskiptoken := -1 // eskiprcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		eskipstate = -1
		eskiprcvr.char = -1
		eskiptoken = -1
	}()
	eskipp := -1
	goto eskipstack

ret0:
	return 0

ret1:
	return 1

eskipstack:
	/* put a state and value onto the stack */
	if eskipDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", eskipTokname(eskiptoken), eskipStatname(eskipstate))
	}

	eskipp++
	if eskipp >= len(eskipS) {
		nyys := make([]eskipSymType, len(eskipS)*2)
		copy(nyys, eskipS)
		eskipS = nyys
	}
	eskipS[eskipp] = eskipVAL
	eskipS[eskipp].yys = eskipstate

eskipnewstate:
	eskipn = eskipPact[eskipstate]
	if eskipn <= eskipFlag {
		goto eskipdefault /* simple state */
	}
	if eskiprcvr.char < 0 {
		eskiprcvr.char, eskiptoken = eskiplex1(eskiplex, &eskiprcvr.lval)
	}
	eskipn += eskiptoken
	if eskipn < 0 || eskipn >= eskipLast {
		goto eskipdefault
	}
	eskipn = eskipAct[eskipn]
	if eskipChk[eskipn] == eskiptoken { /* valid shift */
		eskiprcvr.char = -1
		eskiptoken = -1
		eskipVAL = eskiprcvr.lval
		eskipstate = eskipn
		if Errflag > 0 {
			Errflag--
		}
		goto eskipstack
	}

eskipdefault:
	/* default state action */
	eskipn = eskipDef[eskipstate]
	if eskipn == -2 {
		if eskiprcvr.char < 0 {
			eskiprcvr.char, eskiptoken = eskiplex1(eskiplex, &eskiprcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if eskipExca[xi+0] == -1 && eskipExca[xi+1] == eskipstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			eskipn = eskipExca[xi+0]
			if eskipn < 0 || eskipn == eskiptoken {
				break
			}
		}
		eskipn = eskipExca[xi+1]
		if eskipn < 0 {
			goto ret0
		}
	}
	if eskipn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			eskiplex.Error(eskipErrorMessage(eskipstate, eskiptoken))
			Nerrs++
			if eskipDebug >= 1 {
				__yyfmt__.Printf("%s", eskipStatname(eskipstate))
				__yyfmt__.Printf(" saw %s\n", eskipTokname(eskiptoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for eskipp >= 0 {
				eskipn = eskipPact[eskipS[eskipp].yys] + eskipErrCode
				if eskipn >= 0 && eskipn < eskipLast {
					eskipstate = eskipAct[eskipn] /* simulate a shift of "error" */
					if eskipChk[eskipstate] == eskipErrCode {
						goto eskipstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if eskipDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", eskipS[eskipp].yys)
				}
				eskipp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if eskipDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", eskipTokname(eskiptoken))
			}
			if eskiptoken == eskipEofCode {
				goto ret1
			}
			eskiprcvr.char = -1
			eskiptoken = -1
			goto eskipnewstate /* try again in the same state */
		}
	}

	/* reduction by production eskipn */
	if eskipDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", eskipn, eskipStatname(eskipstate))
	}

	eskipnt := eskipn
	eskippt := eskipp
	_ = eskippt // guard against "declared and not used"

	eskipp -= eskipR2[eskipn]
	// eskipp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if eskipp+1 >= len(eskipS) {
		nyys := make([]eskipSymType, len(eskipS)*2)
		copy(nyys, eskipS)
		eskipS = nyys
	}
	eskipVAL = eskipS[eskipp+1]

	/* consult goto table to find next state */
	eskipn = eskipR1[eskipn]
	eskipg := eskipPgo[eskipn]
	eskipj := eskipg + eskipS[eskipp].yys + 1

	if eskipj >= eskipLast {
		eskipstate = eskipAct[eskipg]
	} else {
		eskipstate = eskipAct[eskipj]
		if eskipChk[eskipstate] != -eskipn {
			eskipstate = eskipAct[eskipg]
		}
	}
	// dummy call; replaced with literal code
	switch eskipnt {

	case 1:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:75
		{
			eskipVAL.routes = eskipDollar[1].routes
			eskiplex.(*eskipLex).routes = eskipVAL.routes
		}
	case 2:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:80
		{
			eskipVAL.routes = []*parsedRoute{eskipDollar[1].route}
			eskiplex.(*eskipLex).routes = eskipVAL.routes
		}
	case 4:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:87
		{
			eskipVAL.routes = []*parsedRoute{eskipDollar[1].route}
		}
	case 5:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:91
		{
			eskipVAL.routes = eskipDollar[1].routes
			eskipVAL.routes = append(eskipVAL.routes, eskipDollar[3].route)
		}
	case 6:
		eskipDollar = eskipS[eskippt-2 : eskippt+1]
//line parser.y:96
		{
			eskipVAL.routes = eskipDollar[1].routes
		}
	case 7:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:101
		{
			eskipVAL.route = eskipDollar[3].route
			eskipVAL.route.id = eskipDollar[1].token
		}
	case 8:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:107
		{
			eskipVAL.token = eskipDollar[1].token
			eskiplex.(*eskipLex).lastRouteID = eskipDollar[1].token
		}
	case 9:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:113
		{
			eskipVAL.route = &parsedRoute{
				matchers:    eskipDollar[1].matchers,
				backend:     eskipDollar[3].backend,
				shunt:       eskipDollar[3].shunt,
				loopback:    eskipDollar[3].loopback,
				dynamic:     eskipDollar[3].dynamic,
				lbBackend:   eskipDollar[3].lbBackend,
				lbAlgorithm: eskipDollar[3].lbAlgorithm,
				lbEndpoints: eskipDollar[3].lbEndpoints,
			}
			eskipDollar[1].matchers = nil
			eskipDollar[3].lbEndpoints = nil
		}
	case 10:
		eskipDollar = eskipS[eskippt-5 : eskippt+1]
//line parser.y:128
		{
			eskipVAL.route = &parsedRoute{
				matchers:    eskipDollar[1].matchers,
				filters:     eskipDollar[3].filters,
				backend:     eskipDollar[5].backend,
				shunt:       eskipDollar[5].shunt,
				loopback:    eskipDollar[5].loopback,
				dynamic:     eskipDollar[5].dynamic,
				lbBackend:   eskipDollar[5].lbBackend,
				lbAlgorithm: eskipDollar[5].lbAlgorithm,
				lbEndpoints: eskipDollar[5].lbEndpoints,
			}
			eskipDollar[1].matchers = nil
			eskipDollar[3].filters = nil
			eskipDollar[5].lbEndpoints = nil
		}
	case 11:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:146
		{
			eskipVAL.matchers = []*matcher{eskipDollar[1].matcher}
		}
	case 12:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:150
		{
			eskipVAL.matchers = eskipDollar[1].matchers
			eskipVAL.matchers = append(eskipVAL.matchers, eskipDollar[3].matcher)
		}
	case 13:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:156
		{
			eskipVAL.matcher = &matcher{"*", nil}
		}
	case 14:
		eskipDollar = eskipS[eskippt-4 : eskippt+1]
//line parser.y:160
		{
			eskipVAL.matcher = &matcher{eskipDollar[1].token, eskipDollar[3].args}
			eskipDollar[3].args = nil
		}
	case 15:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:166
		{
			eskipVAL.filters = []*Filter{eskipDollar[1].filter}
		}
	case 16:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:170
		{
			eskipVAL.filters = eskipDollar[1].filters
			eskipVAL.filters = append(eskipVAL.filters, eskipDollar[3].filter)
		}
	case 17:
		eskipDollar = eskipS[eskippt-4 : eskippt+1]
//line parser.y:176
		{
			eskipVAL.filter = &Filter{
				Name: eskipDollar[1].token,
				Args: eskipDollar[3].args}
			eskipDollar[3].args = nil
		}
	case 19:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:185
		{
			eskipVAL.args = []interface{}{eskipDollar[1].arg}
		}
	case 20:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:189
		{
			eskipVAL.args = eskipDollar[1].args
			eskipVAL.args = append(eskipVAL.args, eskipDollar[3].arg)
		}
	case 21:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:195
		{
			eskipVAL.arg = eskipDollar[1].numval
		}
	case 22:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:199
		{
			eskipVAL.arg = eskipDollar[1].stringval
		}
	case 23:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:203
		{
			eskipVAL.arg = eskipDollar[1].regexpval
		}
	case 24:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:208
		{
			eskipVAL.stringvals = []string{eskipDollar[1].stringval}
		}
	case 25:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:212
		{
			eskipVAL.stringvals = eskipDollar[1].stringvals
			eskipVAL.stringvals = append(eskipVAL.stringvals, eskipDollar[3].stringval)
		}
	case 26:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:218
		{
			eskipVAL.lbEndpoints = eskipDollar[1].stringvals
		}
	case 27:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:222
		{
			eskipVAL.lbAlgorithm = eskipDollar[1].token
			eskipVAL.lbEndpoints = eskipDollar[3].stringvals
		}
	case 28:
		eskipDollar = eskipS[eskippt-3 : eskippt+1]
//line parser.y:228
		{
			eskipVAL.lbAlgorithm = eskipDollar[2].lbAlgorithm
			eskipVAL.lbEndpoints = eskipDollar[2].lbEndpoints
		}
	case 29:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:234
		{
			eskipVAL.backend = eskipDollar[1].stringval
			eskipVAL.shunt = false
			eskipVAL.loopback = false
			eskipVAL.dynamic = false
			eskipVAL.lbBackend = false
		}
	case 30:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:242
		{
			eskipVAL.shunt = true
			eskipVAL.loopback = false
			eskipVAL.dynamic = false
			eskipVAL.lbBackend = false
		}
	case 31:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:249
		{
			eskipVAL.shunt = false
			eskipVAL.loopback = true
			eskipVAL.dynamic = false
			eskipVAL.lbBackend = false
		}
	case 32:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:256
		{
			eskipVAL.shunt = false
			eskipVAL.loopback = false
			eskipVAL.dynamic = true
			eskipVAL.lbBackend = false
		}
	case 33:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:263
		{
			eskipVAL.shunt = false
			eskipVAL.loopback = false
			eskipVAL.dynamic = false
			eskipVAL.lbBackend = true
			eskipVAL.lbAlgorithm = eskipDollar[1].lbAlgorithm
			eskipVAL.lbEndpoints = eskipDollar[1].lbEndpoints
		}
	case 34:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:273
		{
			eskipVAL.numval = convertNumber(eskipDollar[1].token)
		}
	case 35:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:278
		{
			eskipVAL.stringval = eskipDollar[1].token
		}
	case 36:
		eskipDollar = eskipS[eskippt-1 : eskippt+1]
//line parser.y:283
		{
			eskipVAL.regexpval = eskipDollar[1].token
		}
	}
	goto eskipstack /* stack new state and value */
}
