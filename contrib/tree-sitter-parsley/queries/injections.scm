; Language injection queries for Parsley
; Inject CSS grammar into <style> tag raw text content
; Inject JavaScript grammar into <script> tag raw text content

; CSS injection in style tags
(style_tag
  (raw_text) @injection.content
  (#set! injection.language "css")
  (#set! injection.combined))

; JavaScript injection in script tags
(script_tag
  (raw_text) @injection.content
  (#set! injection.language "javascript")
  (#set! injection.combined))
