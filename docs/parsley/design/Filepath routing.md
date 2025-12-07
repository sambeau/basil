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

## What would a site look like?

I was imagining something like this, where:

- the **handler root:** is the same as ever, defined in the config, so no set name: parsley scripts can see everything in it, just like the normal handler can. Basil will not serve anything in the folder, just folders in here defined as public in the config. 
- the **public folder(s):** no change here either, defined in the config, so no set name(s): parsley scripts can see everything in it, just like the normal handler can. Basil will happily server anything in here as a normal file.
- the **site folder**: a new concept, optionally defined in the YAML file. If it is there then Basil will run scripts from paths within this folder. Like other folders, the config will define the path rather than proscribe the name.
- **modules, data, foo**: mo change, these are private folders . Nothing from these file paths will be served by Basil, but Parsley can access them: read, write and execute code.

````Poetry
```tree
handler_root -- as defined in YAML config
handler_root/public -- as defined in YAML config
handler_root/site -- new definition in YAML config
handler_root/modules -- just private folder
handler_root/data -- just private folder
handler_root/foo -- just private folder
:
```
````

## What scripts would be run?

My suggestion is any script called ``index.pars`` will be executed by Basil. When given a path to access, Basil will work from the leaf of the path, towards the root, and execute the first index.pars file it encounters.

So in a path with:
````Poetry
`
/site/reports/2025/Q4/2025.Q4.dat
/site/reports/2025/2025.dat
/site/reports/index.pars
```
````

And given a request for 
- /site/reports/2025/Q4/
Basil will look at:
- /site/reports/2025/Q4/ and reject it as there’s no index
- /site/reports/2025/ again, no index: reject
- /site/reports/ aha! Index file. Execute it! With /2025/Q4/ given in the environment to index.pars
	 
In a path with
````Poetry
`
/site/reports/index.pars
```
````

And given a request for 
- /site/reports/2025/Q4/
Basil will realise there is no /2025/Q4/ so execute /site/reports/index.pars and give /2025/Q4/ to it in the basil environment to do with as it pleases (hopefully we will hand a nicely formatted path object, or similar).

## Filepath Caching?

The decision on caching is performance/implementation driven, so not a design consideration at this point.

## What can index.pars access?
All index.pars files will have access to all the files and folders in the handler root, including (in the example above) modules, data, public etc.

## What if there are two index.pars in a filepath?

This would be normal. It’s always whichever folder is asked for and if it has no index then the path is walked back towards the root until one is found.
e.g.
/index.pars
/pages/home/index.pars
/pages/about/index.pars
/pages/people/index.pars
/pages/people/bob/index.pars
:

If an index is not found, then a 404 is issued. (Very unlikely)

If an index.pars wants to reject a child path, it can explicitly error(404).

There may be other things we can do here to make it less of a chore to check (although an index can just ignore the extra path if it chooses not to look for it).

## Why not simply 404 for a non-existent path?

This structure allows an index file to use the rest of the path as arguments to what files it’s going to serve. In the earlier reports example, for instance, the index.pars file might look in the corresponding data folder looking for a PDF (e.g. /data/reports/2025/Q4/report.pdf)

## What about ?foo=bar&tom=cat etc.?

We’d still have those parsed and in the basil environment too.

