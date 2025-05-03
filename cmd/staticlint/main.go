/*
Package main запускает статический анализатор кода на базе multichecker,
используя набор стандартных, сторонних и пользовательских анализаторов.

## Основной механизм

Программа использует пакет golang.org/x/tools/go/analysis/multichecker
для запуска набора анализаторов (`analysis.Analyzer`) на исходном коде Go.
Механизм multichecker позволяет объединить несколько анализаторов и выполнять
их все сразу при запуске одной команды.

В функцию main добавляются:

1. Все стандартные анализаторы из пакета `go/analysis/passes`.
2. Все SA-анализаторы из пакета `staticcheck` (предупреждения о возможных ошибках).
3. Несколько других специфических анализаторов:
  - QF1001 (предлагает быструю правку)
  - ST1002 и ST1005 (рекомендации по стилю)

4. Пользовательский анализатор `addlint`, запрещающий вызов `os.Exit` в функции `main`.

## Стандартные анализаторы

Эти анализаторы находятся в пакете `golang.org/x/tools/go/analysis/passes`
и обнаруживают типичные ошибки и антипаттерны. Ниже список включённых:

- appends: подозрительное использование append.
- asmdecl: ошибки в объявлениях функций на ассемблере.
- assign: неиспользуемое присваивание.
- atomic: некорректное использование atomic операций.
- bools: избыточные логические выражения.
- buildtag: ошибки в build-тегах.
- cgocall: вызовы C кода в неправильных местах.
- composite: подозрительные литералы композитных типов.
- copylock: передача значений, защищённых мьютексами, по копии.
- defers: defer внутри циклов.
- directive: директивы компиляции с ошибками.
- errorsas: неверное использование errors.As.
- framepointer: отсутствие указателя кадра.
- httpresponse: забытый `Body.Close()` у HTTP-ответов.
- ifaceassert: неверные type assertion на интерфейсы.
- loopclosure: замыкания в цикле на одну переменную.
- lostcancel: потеря вызова `context.CancelFunc`.
- nilfunc: вызов nil-функции.
- printf: ошибки в форматировании строк.
- shift: смещения битов вне допустимого диапазона.
- sigchanyzer: чтение из сигнальных каналов без buffer.
- slog: ошибки в логгировании через `slog`.
- stdmethods: отсутствие стандартных методов (например, String).
- stdversion: проверка соответствия стандартной версии Go.
- stringintconv: преобразование string <-> int с ошибками.
- structtag: неправильные struct-теги.
- testinggoroutine: запуск горутин в тестах без ожидания.
- tests: ошибки в тестах (например, отсутствие `TestXxx`).
- timeformat: ошибки в форматировании времени.
- unmarshal: ошибки при Unmarshal.
- unreachable: недостижимый код.
- unsafeptr: неправильная работа с unsafe.Pointer.
- unusedresult: игнорирование важных возвращаемых значений.
- waitgroup: неправильное использование sync.WaitGroup.

## Staticcheck

Из пакета honnef.co/go/tools/staticcheck включены только анализаторы
с префиксом `SA` — это наиболее строгие проверки, находящие реальные ошибки:

- SA1000–SA4017: широкий диапазон проверок на ошибки и антипаттерны.
- QF1001: быстрая правка — замена strings.ToLower(...)==... на strings.EqualFold.

## Stylecheck

Из пакета honnef.co/go/tools/stylecheck добавлены два анализатора:

- ST1002: проверка именования экспортируемых констант.
- ST1005: проверка сообщений об ошибках — они не должны начинаться с заглавной буквы.

## Пользовательский анализатор: osexitlint

Анализатор `osexitlint` ищет запрещённый вызов `os.Exit(...)` в функции `main`
в пакете `main`.

Пример отчёта:

	os.Exit call is forbidden in main function: os.Exit(1)
*/

package main

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	// Подключение всех стандартных анализаторов из golang.org/x/tools/go/analysis/passes
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/waitgroup"
)

// main собирает все анализаторы и запускает multichecker.
// Используются:
//   - все стандартные анализаторы
//   - все SA-анализаторы (staticcheck)
//   - два ST-анализатора (stylecheck): ST1002 и ST1005
//   - один QF-анализатор: QF1001
//   - собственный анализатор, запрещающий os.Exit в функции main
func main() {
	used := map[string]bool{}
	var analyzers []*analysis.Analyzer

	add := func(a *analysis.Analyzer) {
		if !used[a.Name] {
			analyzers = append(analyzers, a)
			used[a.Name] = true
		}
	}

	// Стандартные анализаторы Go
	analyzers = append(analyzers,
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		waitgroup.Analyzer,
	)

	// SA-группа из staticcheck: серьёзные подозрения на баги
	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			add(a.Analyzer)
		}
	}

	// Дополнительные анализаторы:
	add(staticcheck.Analyzers[50].Analyzer) // QF1001: simplifiable if-return

	add(stylecheck.Analyzers[2].Analyzer) // ST1002: константы должны быть в SCREAMING_SNAKE_CASE
	add(stylecheck.Analyzers[5].Analyzer) // ST1005: ошибки должны начинаться со строчной буквы

	// Кастомный анализатор, запрещающий os.Exit в main
	add(Analyzer)

	multichecker.Main(analyzers...)
}

// render возвращает отформатированное строковое представление AST-узла.
func render(fset *token.FileSet, x interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, x); err != nil {
		panic(err)
	}
	return buf.String()
}

// Analyzer — собственный анализатор, запрещающий вызов os.Exit в функции main.
var Analyzer = &analysis.Analyzer{
	Name:     "osexitlint",
	Doc:      "reports os.Exit",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// run — функция, реализующая проверку: если в функции main найден вызов os.Exit,
// то генерируется сообщение об ошибке. Пропускаются временные файлы, созданные компилятором.
func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}
		if fn.Name.Name != "main" || pass.Pkg.Name() != "main" {
			return
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			obj := pass.TypesInfo.Uses[ident]
			if obj == nil {
				return true
			}

			filename := pass.Fset.File(call.Pos()).Name()
			if strings.Contains(filename, "go-build") {
				return true // игнорировать сгенерированные файлы
			}

			if sel.Sel.Name == "Exit" {
				if pkgObj, ok := pass.TypesInfo.Uses[ident].(*types.PkgName); ok && pkgObj.Imported().Path() == "os" {
					pass.Reportf(call.Pos(), "os.Exit call is forbidden in main function: %s", render(pass.Fset, call))
				}
			}

			return true
		})
	})

	return nil, nil
}
