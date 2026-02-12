; Bracket pairs for Parsley
("[" @open "]" @close)
("{" @open "}" @close)
("(" @open ")" @close)

; Tags - exclude from rainbow brackets
(("<" @open ">" @close) (#set! rainbow.exclude))

; Strings - exclude from rainbow brackets
(("\"" @open "\"" @close) (#set! rainbow.exclude))

; Query DSL - interpolation braces
(query_interpolation
  "{" @open
  "}" @close)

; Query DSL - condition groups
(query_condition_group
  "(" @open
  ")" @close)
