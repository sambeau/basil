---
title: Why?
---

## Who made this?

✉️ [Sam Phillips](mailto:sambeau@mac.com) made this. 

I am an unemployed design/product manager, currently based in Scotland (working on a [replacement for Facebook ](https://Tickly.org)). I started out as a computer scientist, then became a full stack developer, then a designer. Now I’m unemployed—don't look for a job in your 50s, as the AIs reject you out-of-hand.

## Why did I make this?

Just for fun, really.

I guess there was an element of trying to go back to a simpler time when making websites was quick and easy; partly I wanted to see if all my computer language ideas could work together; partly I was trying to convince myself that agentic development was the load of nonsense. Spoiler: I was very wrong; it's not.

I joke that this is "Making Websites Like it's 2002" and there's a grain of truth to that. Back in the early 2000s you could drop a Perl file in a folder and have a dynamic website. The only security worries you had was SQL injection. Being a developer was easy. Now, for better or worse, there's a ton of decisions to be made, packages to install and constant upgrades to maintain. Basil is an attempt to get away from this. But I confess I may have rose-tinted glasses about the 'good old days'.

## A little history

Back in 2001, just after the dot-com bubble was burst, I was in Cambridge and deep in grief after the death of my son. To keep busy, I built a small concatenated language called ‘Basil’ that I used to build a number of commercial websites.

I’d studied Programming Languages under the Glasgow Haskell team, so I built an almost-but-not-quite *pure*, almost-but-not-quite *functional* language where every expression returned HTML. It was far more expressive than the other languages at the time (including Perl) and being that all code was also HTML it made it extremely speedy to build sites with.

Ever since then I’ve been meaning to return to it—there’s a half-written C version in my GitHub from around 2005. Then, JSX came along and I jealously thought, “I kind’a built that shit back in 2001”.

One day in November 2025, I thought I’d better find out what the fuss about Claude was. So, I tried to build a simple parser with it, trying to prove to myself LLMs were nonsense and there was no way they could build anything useful *＊cough＊.* Before long, I’d built a modern version of my old language. I called it Parsley as a tribute to Basil *(and, because …’parse’)*.

Within a few weeks I was knee-deep in design documents and specifications. A few months of solid work later I had built a mature and expressive language, and soon after a super-fast server to run it on (which I called ‘Basil’ in tribute to Basil). It was a lot of work, but nowhere near the years of work it would have taken me to code it by myself. Plus, I’d had a lot of fun playing with new ideas I’d been thinking about for years: fun types (hat tip to [Carl Sassenrath and Rebol](https://www.rebol.com)), Parts, Tables, Records, Schema, Database DSL, Measurement Units. 

And, I am rather proud of it.
 
## Why should you try this?

It’s fun. And, once you get to know them, Parsley and Basil feel expressive and frictionless. They do all the boring stuff for you, including all the tedious security stuff, and they make all the fun stuff easy. If you still like the idea of making something by hand, using your own brain, then give them a try. You might even end up liking their quirks.

Also, I don’t mean to shit on NPM—lots of brilliant people have put a loads of work into them—but, if like me you are tired of all the donkey-work needed to just get a React site up and running, then maybe you’ll enjoy a single-file install?

## Why shouldn’t you try this?

Two reasons:

1. Modern tools like Typescript and React are mature and really good at building websites. Python and Go are brilliant for munging data and creating CLI-tools. Why learn something new and quirky?
2. LLMs have devoured a shit-ton of Typescript and Javascript code, so are *really* good at writing them. There is no Parsley code out there, so they write it *very badly* even with good docs and a skill (take a look at the script that built this). Why use a new language to build a website when you can ask Claude to build you one in an hour?

Plus, **Parsley doesn’t have types**. I will claim that it doesn’t need them and that Schema are enough, but you will probably disagree with me.

## So, was this ‘vibe-coded’?

*Ha, **no**.*

This took months to create. Last I measured (July 2026) there were: 1,297 commits, 149 feature specification documents, 201,344 lines of code (100,797 lines of implementation and 100,547 lines of test code), 2,611 tests in 243 files, 619 Markdown files, 992,387 words written of which 781,072 words are workflow docs and 175,868 words are user-facing documentation.

Of course, [lines of code is a stupid metric](https://www.folklore.org/Negative_2000_Lines_Of_Code.html), I only mention them to show this project was a lot of work,  carefully nurtured over 9 months, with a human making all the important decisions.

## Thank, you

Thanks for taking a look. I'll happily take feedback if you have any, especially anything security related. But, no, I'm very unlikley to add type-checking.

Made in 🏴󠁧󠁢󠁳󠁣󠁴󠁿 with ❤️.
