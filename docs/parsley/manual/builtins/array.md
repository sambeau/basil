---
id: array
title: Arrays
system: parsley
type: builtin
name: array
created: 2025-12-16
version: 1.0.0
author: Basil Team
keywords:
  - array
  - list
  - collection
  - sequence
  - iteration
  - sort
---

# Arrays

Arrays are ordered collections of values of any type. They form the foundation of data processing in Parsley, supporting indexing, slicing, functional operations (map, filter, reduce), and convenient methods for sorting, formatting, and random selection.

## Literals

Create arrays using square brackets with comma-separated elements:

```parsley
[1, 2, 3]
```

**Result:** `[1, 2, 3]`

Arrays can contain any mix of types:

```parsley
[1, "hello", true, null, £5.00]
```

**Result:** `[1, "hello", true, null, £5.00]`

Create an empty array:

```parsley
[]
```

**Result:** `[]`

Arrays can be nested:

```parsley
[[1, 2], [3, 4], [5, 6]]
```

**Result:** `[[1, 2], [3, 4], [5, 6]]`

## Operators

### Indexing

Access elements by position using bracket notation. Positions are zero-indexed:

```parsley
cities = ["London", "Paris", "Tokyo"]
cities[0]
```

**Result:** `"London"`

Access from the end using negative indices. The last element is `-1`:

```parsley
cities[-1]
```

**Result:** `"Tokyo"`

```parsley
cities[-2]
```

**Result:** `"Paris"`

Use optional indexing to safely access out-of-bounds positions (returns `null`):

```parsley
cities[?10]
```

**Result:** `null`

### Slicing

Extract a range of elements using slice notation `[start:end]`. The slice is inclusive of start and exclusive of end:

```parsley
numbers = [10, 20, 30, 40, 50]
numbers[1:3]
```

**Result:** `[20, 30]`

Slice from the beginning:

```parsley
numbers[:2]
```

**Result:** `[10, 20]`

Slice to the end of the array:

```parsley
numbers[2:]
```

**Result:** `[30, 40, 50]`

### Concatenation

Combine arrays with the `++` operator:

```parsley
[1, 2] ++ [3, 4]
```

**Result:** `[1, 2, 3, 4]`

Concatenate multiple arrays:

```parsley
["a"] ++ ["b", "c"] ++ ["d"]
```

**Result:** `["a", "b", "c", "d"]`

### Repetition

Repeat an array using the `*` operator:

```parsley
[1, 2] * 3
```

**Result:** `[1, 2, 1, 2, 1, 2]`

## Methods

### filter

Returns a new array containing only elements where the predicate function returns true:

```parsley
[1, 2, 3, 4, 5].filter(fn(x) { x > 2 })
```

**Result:** `[3, 4, 5]`

Filter using ``for``:

```parsley
for(x in [1, 2, 3, 4, 5]) { if(x > 2){x} }
```

**Result:** `[3, 4, 5]`

Filter strings:

```parsley
["apple", "banana", "apricot"].filter(fn(s) { s[0] == "a" })
```

**Result:** `["apple", "apricot"]`

### format

Format the array as a readable list in the user's locale. Supports styles: `"and"`, `"or"`, and `"unit"`:

```parsley
["Alice", "Bob", "Charlie"].format("and")
```

**Result:** `"Alice, Bob, and Charlie"`

Using `"or"` style:

```parsley
["coffee", "tea", "milk"].format("or")
```

**Result:** `"coffee, tea, or milk"`

Using `"unit"` style:

```parsley
[1, 2, 3].format("unit")
```

**Result:** `"1, 2, 3"`

In French:

```parsley
["Alice", "Bob", "Charlie"].format("and","Fr")
```

**Result:** `"Alice, Bob et Charlie"`

### insert

Insert an element at a specific index, returning a new array. Supports negative indices:

```parsley
["a", "c"].insert(1, "b")
```

**Result:** `["a", "b", "c"]`

Insert at the beginning with index 0:

```parsley
[2, 3].insert(0, 1)
```

**Result:** `[1, 2, 3]`

Insert at the end using index equal to array length:

```parsley
[1, 2].insert(2, 3)
```

**Result:** `[1, 2, 3]`

Insert using negative index (inserts *before* that position):

```parsley
["a", "b"].insert(-1, "x")
```

**Result:** `["a", "x", "b"]`

### join

Concatenate all elements into a single string, separated by a delimiter:

```parsley
["Hello", "world"].join(" ")
```

**Result:** `"Hello world"`

Join with empty string:

```parsley
["a", "b", "c"].join("")
```

**Result:** `"abc"`

Join numbers:

```parsley
[1, 2, 3].join("-")
```

**Result:** `"1-2-3"`

### length

Get the number of elements in the array:

```parsley
[10, 20, 30].length()
```

**Result:** `3`

Empty array:

```parsley
[].length()
```

**Result:** `0`

### map

Transform each element using a function. Returns a new array with the transformed elements:

```parsley
[1, 2, 3].map(fn(x) { x * 2 })
```
**Result:** `[2, 4, 6]`

The same, buty using ``for`` style:

```parsley
for(x in [1,2,3]){x * 2}
```

**Result:** `[2, 4, 6]`

Extract properties from objects:

```parsley
users = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
users.map(fn(u) { u.name })
```

**Result:** `["Alice", "Bob"]`

### pick

Select random elements from the array *with replacement* (same element can be picked multiple times):

```parsley
[1, 2, 3].pick(2)
```

**Result:** Two random elements (may include duplicates), 
e.g. ⚡️<code>@{repr([1, 2, 3].pick(2))}</code>

Pick from a list of options:

```parsley
["red", "green", "blue"].pick(1)
```

**Result:** An array of one random color, e.g. 
⚡️<code>@{repr(["red", "green", "blue"].pick(1))}</code>

### reduce

Accumulate values using a function that takes the accumulator and current element, starting from an initial value:

```parsley
[1, 2, 3, 4].reduce(fn(sum, x) { sum + x }, 0)
```

**Result:** `10`

Build a string:

```parsley
["a", "b", "c"].reduce(fn(str, x) { str ++ x }, "")
```

**Result:** `"abc"`

Sum prices:

```parsley
[£10.50, £5.25, £12.00].reduce(fn(total, price) { total + price }, £0.00)
```

**Result:** `£27.75`

### reorder

Apply `.reorder()` to each dictionary element in the array. Returns a new array where each dictionary has been reordered (and optionally renamed).

**With an array argument** — select and reorder keys in each dictionary:

```parsley
let users = [{name: "Alice", age: 30, city: "London"}, {name: "Bob", age: 25, city: "Paris"}]
users.reorder(["city", "name"])
```

**Result:** `[{city: "London", name: "Alice"}, {city: "Paris", name: "Bob"}]`

**With a dictionary argument** — rename and reorder keys:

```parsley
let data = [{first_name: "Alice", last_name: "Smith"}, {first_name: "Bob", last_name: "Jones"}]
data.reorder({name: "first_name", surname: "last_name"})
```

**Result:** `[{name: "Alice", surname: "Smith"}, {name: "Bob", surname: "Jones"}]`

This is useful for:
- Preparing data for display or export
- Renaming columns from a database or API to match your template
- Selecting a subset of fields from each row

**Note:** Non-dictionary elements in the array are left unchanged.

### reverse

Reverse the order of elements:

```parsley
[1, 2, 3].reverse()
```

**Result:** `[3, 2, 1]`

Reverse strings:

```parsley
["first", "second", "third"].reverse()
```

**Result:** `["third", "second", "first"]`

### shuffle

Randomly shuffle the array using the Fisher-Yates algorithm:

```parsley
[1, 2, 3, 4, 5].shuffle()
```

**Result:** Array with elements in random order
e.g. ⚡️<code>@{repr( [1, 2, 3, 4, 5].shuffle() )}</code>


Shuffle a list of names:

```parsley
["Alice", "Bob", "Charlie", "Diana"].shuffle()
```

**Result:** Names in random order
e.g. ⚡️<code>@{repr( ["Alice", "Bob", "Charlie", "Diana"].shuffle() )}</code>

### sort

Sort the array in natural order (numbers ascending, strings alphabetically):

```parsley
[3, 1, 4, 1, 5, 9].sort()
```

**Result:** `[1, 1, 3, 4, 5, 9]`

Sort strings:

```parsley
["banana", "apple", "cherry"].sort()
```

**Result:** `["apple", "banana", "cherry"]`

⚠️ Sort uses [Natural sort](https://en.wikipedia.org/wiki/Natural_sort_order):

```parsley
["10 banana", "9 apple", "100 cherry"].sort()
```

**Result:** `["1 apple", "10 banana", "16 cherry"]`

### sortBy

Sort the array by a derived key function:

```parsley
users = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
users.sortBy(fn(u) { u.age })
```

**Result:** `[{name: "Bob", age: 25}, {name: "Alice", age: 30}]`

Sort by string length:

```parsley
["hello", "a", "goodbye"].sortBy(fn(s) { s.length() })
```

**Result:** `["a", "hello", "goodbye"]`

Sort is stable:

```parsley
users = [{name: "Bob", age: 30}, {name: "Alice", age: 25},
         {name: "Bob", age: 40}, {name: "Alice", age: 50}]
users.sortBy(fn(u) { u.age }).sort()
```

**Result:** `[{name: Alice, age: 25}, {name: Alice, age: 50}, {name: Bob, age: 30}, {name: Bob, age: 40}]`


### take

Select random elements from the array *without replacement* (each element picked at most once):

```parsley
[1, 2, 3, 4, 5].take(3)
```

**Result:** Three distinct random elements (no duplicates) 
e.g. ⚡️<code>@{repr( [1, 2, 3, 4, 5].take(3) )}</code>


Deal cards from a deck:

```parsley
suits = ["♠", "♥", "♦", "♣"]
suits.take(2)
```

**Result:** Two random distinct suits, 
e.g. ⚡️<code>@{repr( ["♠", "♥", "♦", "♣"].take(2) )}</code>


### toCSV

Convert the array to CSV format (with proper quoting and escaping):

```parsley
[["Name", "Age"], ["Alice", 30], ["Bob", 25]].toCSV()
```

**Result:** CSV string with proper formatting

```csv
  Name,Age
  Alice,30
  Bob,25
```
Simple array:

```parsley
["John", "Jane", "Jack"].toCSV()
```

**Result:** `"John,Jane,Jack"`

### toJSON

Convert the array to JSON format:

```parsley
[1, 2, {"name": "Alice"}].toJSON()
```

**Result:** 
```json
[1, 2, {"name": "Alice"}].toJSON()
```

JSON table:

```parsley
[{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}].toJSON()
```

**Result:**
```json
[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]
```

### toBox

Render the array in a box with box-drawing characters. Useful for CLI output and debugging:

```parsley
["apple", "banana", "cherry"].toBox()
```

**Result:**

```
┌────────┐
│ apple  │
├────────┤
│ banana │
├────────┤
│ cherry │
└────────┘
```

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `direction` | string | `"vertical"` | Layout: `"vertical"`, `"horizontal"`, `"grid"` |
| `align` | string | `"left"` | Text alignment: `"left"`, `"right"`, `"center"` |
| `style` | string | `"single"` | Box border style: `"single"`, `"double"`, `"ascii"`, `"rounded"` |
| `title` | string | none | Title row centered at top of box |
| `maxWidth` | integer | none | Truncate content to this width (adds `...`) |

#### Direction Examples

Horizontal layout:

```parsley
[1, 2, 3].toBox({direction: "horizontal"})
```

**Result:**

```
┌───┬───┬───┐
│ 1 │ 2 │ 3 │
└───┴───┴───┘
```

Grid layout (auto-detected for array of arrays):

```parsley
[[1, 2, 3], [4, 5, 6]].toBox()
```

**Result:**

```
┌───┬───┬───┐
│ 1 │ 2 │ 3 │
├───┼───┼───┤
│ 4 │ 5 │ 6 │
└───┴───┴───┘
```

#### Style Examples

```parsley
["A", "B", "C"].toBox({style: "double", direction: "horizontal"})
```

**Result:**

```
╔═══╦═══╦═══╗
║ A ║ B ║ C ║
╚═══╩═══╩═══╝
```

#### Title Example

```parsley
[1, 2, 3].toBox({title: "Numbers"})
```

**Result:**

```
┌─────────┐
│ Numbers │
├─────────┤
│    1    │
├─────────┤
│    2    │
├─────────┤
│    3    │
└─────────┘
```

## Examples

### Processing a List of Numbers

Calculate the sum and average of prices:

```parsley
prices = [£10.00, £15.50, £8.25, £12.75]
total = prices.reduce(fn(sum, p) { sum + p }, £0.00)
average = total / prices.length()
```

**Result:** `average = £11.63`

### Filtering and Transforming Data

Extract names of people over 25 and sort alphabetically:

```parsley
people = [
  {name: "Alice", age: 30},
  {name: "Charlie", age: 22},
  {name: "Bob", age: 28}
]
adults = people
  .filter(fn(p) { p.age > 25 })
  .map(fn(p) { p.name })
  .sort()
```

**Result:** `["Alice", "Bob"]`

### Random Selection

Pick a random Jedi for the mission:

```parsley
jedi = ["Yoda", "Luke", "Leia", "Obi-Wan", "Mace"]
chosen = jedi.pick(1)
```

**Result:** One randomly selected Jedi name,
e.g. <code>chosen = @{repr( ["Yoda", "Luke", "Leia", "Obi-Wan", "Mace"].pick(1) )}</code>

Select a team of 3 distinct Avengers:

```parsley
avengers = ["Iron Man", "Captain America", "Thor", "Black Widow", "Hawkeye"]
team = avengers.take(3)
```

**Result:** Three distinct random Avengers,
e.g. <code>chosen = @{repr( ["Iron Man", "Captain America", "Thor", "Black Widow", "Hawkeye"].take(3) )}</code>


### Array Manipulation

Chunk an array into groups:

```parsley
items = [1, 2, 3, 4, 5, 6, 7, 8, 9]
pairs = items / 2
```

**Result:** Groups of 2 elements (last group may be smaller),
`[[1, 2], [3, 4], [5, 6], [7, 8], [9]]`

Combine multiple lists:

```parsley
breakfast = ["eggs", "toast", "bacon"]
lunch = ["sandwich", "fruit"]
dinner = ["pasta", "salad", "bread"]
meals = breakfast ++ lunch ++ dinner
```

**Result:** `["eggs", "toast", "bacon", "sandwich", "fruit", "pasta", "salad", "bread"]`

### Counting and Summarizing

Count specific items:

```parsley
votes = ["Alice", "Bob", "Alice", "Charlie", "Alice", "Bob"]
alice_votes = votes.filter(fn(v) { v == "Alice" }).length()
```

**Result:** `3`

Format a list for presentation:

```parsley
winners = ["Alice", "Bob", "Charlie"]
announcement = "Congratulations to " ++ winners.format("and") ++ "!"
```

**Result:** `"Congratulations to Alice, Bob, and Charlie!"`

In German:

```parsley
winners = ["Anna", "Bernd", "Claudia"]
announcement = "Herzlichen Glückwunsch " ++ winners.format("and", "DE") ++ "!"
```

**Result:** `Herzlichen Glückwunsch an Alice, Bob und Charlie!`

### Advanced: Custom Sorting

Sort movies by release year, then by title:

```parsley
movies = [
  {title: "The Force Awakens", year: 2015},
  {title: "A New Hope", year: 1977},
  {title: "The Last Jedi", year: 2017}
]
sorted = movies
  .sortBy(fn(m) { m.year })
  .sortBy(fn(m) { m.title })
```

**Result:** Movies sorted by year, then alphabetically within each year

### Random Shuffle with Sampling

Shuffle a deck and deal cards:

```parsley
deck = ["2♠", "3♠", "4♠", "5♠", "6♠", "7♠", "8♠", "9♠", "10♠", "J♠", "Q♠", "K♠", "A♠"]
hand = deck.shuffle().take(5)
```

**Result:** Five random distinct cards from the shuffled deck,
e.g. ⚡️<code>@{repr( ["2♠", "3♠", "4♠", "5♠", "6♠", "7♠", "8♠", "9♠", "10♠", "J♠", "Q♠", "K♠", "A♠"].shuffle().take(5) )}</code>
