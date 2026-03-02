# Integration tools

As part of the push to Basil v1.0, I think we should consider building and/or installing some integration testing tools.

There are parts of Basil that cannot easily be tested: SSH, Database drivers, Git server, SFTP, Fetch, email, etc.

I am wonder how we could automate some of this testing? In particular, I am looking for strategies for AI agents to test as they code.

There are various strategies:

- Mocking them
- A live web site/servers (HTTPS, Git, SFTP etc)
- A suite of servers
- Accounts with 3rd party suppliers (Supabase, Email providers etc)
- Others?

These could also be
- A docker container
- Just servers/tools that run
- Go programs based around already-existing libraries

The simpler the solution the better, so a local tool that can just be run is *always* preferable.

Can we build some of these servers? Should we?

## Questions
- What is the best strategy?
- Do we need a different strategy for each?
- What are the advantages/disadvantages of a Docker container?
- What are the advantages/disadvantages of a live web site/servers?
- How many of these servers can we build ourselves and do we want to?

AS usual we are looking for a simple solution that is “good enough”.
