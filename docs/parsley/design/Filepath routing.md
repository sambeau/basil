# Filepath routing

Basil is has tried to offer two way to do things: basic (batteries-included, that will cover most use-cases) and expert (which lets you bring your own batteries).

So to be batteries-included, Basil needs a router.

There are two options:
- build something like we have for std/API
- build something super-simple

I have chosen to go the simple route: something familiar to most  developers that will be simple to understand: a filepath router.

- **Beginners:** route by file
- **Experts:** one handler, roll-your own router

Filepath routing is what web servers have done since web servers were invented, and I’m not proposing to do anything radically different from that, apart from a minor tweak or two to make it integrate well with Parsley.

