---
title: Why?
---

## Why did I make this?

Partly, just for fun; partly because I feel that we’ve lost something of the early days of websites that I wanted to put back. *Oh, … and I wanted to prove that LLMs were a bag of shite.*

Back in 2001, just when the dot-com bubble was bursting, I was deep in grief after the death of my son. To keep busy, I built a small concatenated language called ‘Basil’ that I used to build a number of commercial websites. I’d studied Programming Languages under the Glasgow Haskell team, so I built an almost-but-not-quite *pure*, almost-but-not-quite *functional* language where every expression returned HTML. It was far more expressive than the other languages at the time (including Perl) and being that all code was also HTML it made it extremely speedy to build sites with.

Ever since then I’ve been meaning to return to it—there’s a half-written C version in my GitHub from around 2005. Then, JSX came along and I jealously thought, “I kind’a built that shit back in 2001”.

One day in November 2025, I thought I’d better find out what the fuss about Claude was. So, I tried to build a simple parser with it, trying to prove to myself LLMs were nonsense and there was no way they could build anything useful. *＊cough＊* Before long, I’d built a modern version of my old language. I called it Parsley as a tribute to Basil *(and because …’parse’ geddit?)*.

Within a few weeks I was knee deep in design documents and specifications. A few months of solid work later I had built a mature and expressive language, and soon after a super-fast server to run it on (which I called ‘Basil’ in tribute to Basil). It was a lot of work, but nowhere near the years of work it would have taken me to code it by myself. Plus, I’d had a lot of fun playing with new ideas I’d been thinking about for years: fun types (hat tip to [Carl Sassenrath and Rebol](https://www.rebol.com)), Parts, Tables, Records, Schema, Database DSL, units.

And I am rather proud of it all.
 
 *(Turns out—no—LLMs aren’t a bag of shite. At least not for coding)*

## Why should you try this?

It’s fun and, once you get to know them, Parsley and Basil feel expressive and frictionless. They do all the boring stuff for you and make all the fun stuff easy. If you still like the idea of making something by hand, using your own brain, then give them a try. You might even end up liking some of the quirks.

Also, I don’t mean to shit on NPM—lots of brilliant people have put a buttload of work into it—but, if, like me, you are tired of all the donkey-work needed to just get a React site up and running, then maybe you’ll enjoy a single-file install.

## Why shouldn’t you try this?

Two reasons:

1. Modern tools like Typescript and React are mature and really good at building websites. Python and Go are brilliant for munging data and creating CLI-tools. Why learn something new and quirky?
2. LLMs have devoured a shit-ton of Typescript and Javascript code, so are *really* good at writing them. There is no Parsley code out there, so they write it *very badly* even with good docs and a skill. Why use a new language to build a website when you can ask Claude to build you one in an hour?

Plus, Parsley doesn’t have types. I will claim that it doesn’t need them and that Schema are enough, but you will probably disagree with me.

## So, was this ‘vibe-coded’?

*Ha, **no**.*

This took months to create. As of now we are at: 1,297 commits, 149 feature specification documents, 201,344 lines of code (100,797 lines of implementation and 100,547 lines of test code), 2,611 tests in 243 files, 619 Markdown files, 992,387 words written of which 781,072 words are workflow docs and 175,868 words are user-facing documentation.

Of course, [lines of code is a stupid metric](https://www.folklore.org/Negative_2000_Lines_Of_Code.html), but this project was a lot of work,  carefully nurtured over half a year, with a human making all the important decisions.
