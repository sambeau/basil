# Public files

When I develop sites I like to keep some non-script files in the same directory as the module—usually user-interface elements like SVG files for a component. While SVG embedding is an option, it isn’t always the right option: browsers treat SVG files differently to how they treat embedded SVG (e.g. caching). Having a file in a folder beside the component’s module code also encapsulates a Component as a single folder: easy to reason about; easy to reuse.

This is common in the world of javascript compilers.

So the question is, what is the best way for Basil—which up until now has a strict enforcement of what is public and private—accommodate this without breaking that boundary.

I see two main options, but we should investigate how the javascript compilers do it.
1. ‘Bless’ single files to let them be public using a builtin standard library function.
2. Copy the file into the public directory and return a file id/URL that is then used to reference it in the code

Are there other ways to do this?

We are of course looking for the most simple, minimal, composable method.

I would classify this in the ‘more-than-basic’ features: the basic use-case that most inexperienced developers will use is the simple: ‘stick-it-in-the-public-folder’ method.

## ‘Bless’ single files
Without straying too far into spec and implementation, this option would be a special path operation that somehow informs the server that this file is okay to send to the client. Somewhere there is a list of exceptions to the public/private rule and if it is on the list it can be sent.

It does leak information about file structure—module paths and names would appear in the url (assuming a naive approach to the URL). Plus it would mean keeping a cache of ‘blessed’ files.

## Copy + id/URL
This essentially moves the cache, physically into the public folder. When and how is not clear to me: when a javascript compiler does this, it’s done at the point of compilation.

## A combination?
Could we obfuscate the URL path as part of the caching process and return this to the module code to use as a reference, the rewrite on-the-fly?

## Questions
- Is this just a special form of caching? (See FEAT-037)
- does the javascript world have an elegant solution to this?
- does anyone do this ‘live’ rather than as a compilation step?

Any thoughts, suggestions?