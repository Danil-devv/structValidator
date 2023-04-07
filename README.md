# structValidator

## Описание

Пакет реализует функцию для валидации публичных полей входной структуры на основе структурного тэга `validate`.

```go
func Validate(v any) error
```

Функция возвращает:
- nil, если структура валидна;
- или ошибку, произошедшую во время валидации;
- или `ValidationErrors` - ошибку с информацией, содержащей имя поля и ошибку его валидации (ошибки валидации накапливаются).

Типы поддерживаемых полей:
- `int`, `[]int`, `[C]int`;
- `string`, `[]string`, `[C]string`.

Если передан слайс или массив, то каждый его элемент проверяется согласно спецификации в теге;

Поддерживаемые валидаторы:

* `len:10` - [string] длина строки 10 символов;
* `in:val1,val2`
    * [string] вхождение в {string, string, ... };
    * [int]  вхождение в {int, int, ... };
* `min:10`
    * [string] минимальная длинна строки;
    * [int] >= 10;
* `max:20`
    * [string] максимальная длинна строки;
    * [int] <= 20;

Валидаторы можно совмещать, для этого используется разделитель `;`

## Примеры использования

```go
v: struct {
    InInt  [2]int   `validate:"in:20,25,30"`
    Len    []string `validate:"len:20"`
    InStr  []string `validate:"in:foo,bar"`
    MinMaxInt int    `validate:"min:10;max:30"`
    LenStr []string `validate:"min:10;max:100"`
}{
    Len:    []string{"abcdefghjklmopqrstvu", "abcdefghjklmopqrstvu"},
    InInt:  [2]int{25, 20},
    InStr:  []string{"bar", "foo"},
    MinMaxInt: 15,
    LenStr: []string{"abcdefghjkl", "abcdefghjklmnopqrst"},
},
```