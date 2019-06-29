---
layout: post
title: Architecture Decision Records
tags: architecture process design
---

This is a text version of a short talk (affectionately known as a "Coffee Bag") I gave at work this week, on Architecture Design Records.  You can see the [slides here](/presentations/index.html?adr), but there isn't a recording available, unfortunately.

It should be noted; these are not to replace full architecture diagrams; you should definitely still write [C4 Models](https://c4model.com) to cover the overall architecture.  ADRs are for the details, such as serializer formats, convention-over-configuration details, number precisions for timings, or which metrics library is used and why.

## What?

Architecture Design Records are there to solve the main question people repeatedly ask when they view a new codebase or look at an older part of their current codebase:

> Why on earth was it done like this?!

Generally speaking, architectural decisions have been made in good faith at the time, but as time marches on, things change, and the reasoning gets lost.  The reasoning might be discoverable through the commit history, or some comments in a type somewhere, and every once in a while, people remember the Wiki exists, and hope that someone else remembered and put some docs there.  They didn't by the way.

Architecture Design Records are aiming to solve all of this, with three straightforward attributes: Easy to Write, Easy to Read, and Easy to Find.  Let's look at these on their own, and then have a look at an example.

## Easy to Find

As I alluded to earlier, "easy to find" doesn't mean "hidden in confluence" (or any other wiki, for that matter.)  The best place to put records of architecture decisions is in the repository.  If you want them elsewhere, that's fine, but the copy in the repository should be the source of truth.

As long as the location is consistent (and somewhat reasonable), it doesn't matter too much where they go.  I like to put them in the `docs/arch` path, but a common option is `docs/adr` too:

```bash
$ tree ~/dev/projects/awesome-api
|-- docs
|   `-- arch
|       |-- api-error-codes.md
|       |-- controller-convention.md
|       `-- serialization-format.md
|-- src
|-- test
`-- readme.md
```

The file names for each architecture decision are imperative - e.g. "serialization format", rather than "figure out what format to use", much like your commit messages are (right?)  You might also note that the files are Markdown.  Because what else would they be really?

## Easy to Write

As just mentioned, I usually use Markdown for writing all documents, but as long as you are consistent (notice a pattern here?) and that it is plain-text viewable (i.e. in a terminal), it doesn't matter too much.  Try and pick a format that doesn't add much mental overhead to writing the documents, and if it can be processed by tools easily, that's a bonus, as we will look into later.

## Easy to Read

There are two components to this:  Rendering and Format.

Rendering is covering how we actually read it - plain text in a terminal, syntax highlighting in an editor, or rendered into a web page.  Good ADRs can handle all three, and Markdown is a good fit for all of them!  By using Markdown, not only can we render to HTML, we can even use Confluences's questionable "Insert Markdown Markup" support to write them into a wiki location if desired.

Format is covering what the content of the document is.  There are [many different templates you can use](https://github.com/joelparkerhenderson/architecture_decision_record), which have different levels of detail, and are aimed at different levels of decisions.  I like to use a template based off [Michael Nygard's](https://github.com/joelparkerhenderson/architecture_decision_record/blob/master/adr_template_by_michael_nygard.md), which I modified a little bit to have the following sections:

* Title
* Status
* Context
* Considered Options
* Chosen Decision
* Consequences

Let's have a look at these in an example.

## Example

We have a new API we are developing, and we need to figure out which serialization format we should use for all the requests and responses it will handle.

We'll start off with our empty document and add in the Title, and Status:


```markdown
# Serialization Format

## Status

In Progress
```

The Title is *usually* the same as the file name, but not necessarily.  The Status indicates where the document is in its lifespan.  What statuses you choose is up to you, but I usually have:

* In Progress
* Accepted
* Rejected
* Superseded
* Deprecated

Once an ADR is Accepted (or Rejected), the content won't change again.  Any subsequent changes will be a new ADR, and the previous one will be marked as either Deprecated or Superseded, along with a link to the ADR which replaces it, for example:


```markdown
## Status

Superseded by [Api Transport Mechanisms](api-transport-mechanisms.md)
```

Next, we need to add some context for the decision being made.  In our serialization example, this will cover what area of the codebase we are covering (the API, rather than storage), and any key points, such as message volume, compatibilities etc.

```markdown
## Context

We need to have a consistent serialization scheme for the API.  It needs to be backwards and forwards compatible, as we don't control all of the clients.  Messages will be fairly high volume and don't *need* to be human readable.
```

Now that we have some context, we need to explain what choices we have available.  This will help when reading past decisions, as it will let us answer the question "was xxxx or yyyy considered?".  In our example, we consider JSON, Apache Avro, the inbuilt binary serializer, and a custom built serializer (and others, such as Thrift, ProtoBufs, etc.)


```markdown
## Considered Options

1. **Json**: Very portable, and with serializers available for all languages.  We need to agree on a date format, and numeric precision, however.  The serialization should not include white space to save payload size.  Forwards and Backwards compatibility exists but is the developer's responsibility.

2. **Apache Avro**: Binary format which includes the schema with the data, meaning no need for schema distribution.  No code generator to run, and libraries are available for most languages.

3. **Inbuilt Binary**: The API is awkward to use, and its output is not portable to other programming languages, so wouldn't be easy to consume for other teams, as well as some of our internal services.

4. **Custom Built**: A lot of overhead for little to no benefit over Avro/gRPC etc.

5. **Thrift**: ...
```

The second to last section is our Chosen Decision, which will not only list which one we picked (Avro, in this case) but also why it was chosen over other options.  All this helps reading older decisions, as it lets you know what was known at the time the decision was made - and you will always know less at the time of the decision than you do now.

```markdown
## Chosen Decision

**2. Apache Avro**

Avro was chosen because it has the best combination of message size and schema definition.  No need to have a central schema repository set up is also a huge benefit.
```

In this example, we have selected Avro and listed that our main reasons were message size, and the fact that Avro includes the schema with each message, meaning we don't need a central (or distributed) schema repository to be able to read messages.

The final section is for Consequences of the decision.  This is **not** to list reasons that we could have picked other decisions, but to explain things that we need to start doing or stop doing because of this decision.  Let's see what our example has:

```markdown
## Consequences

As the messages are binary format, we cannot directly view them on the wire.  However, a small CLI will be built to take a message and pretty print it to aid debugging.
```

As we have selected a binary message format, the messages can't be easily viewed any more, so we will build a small CLI which when given a message (which as noted, contains the schema), renders a human-readable version of the message.

## Dates

You might notice that the record doesn't contain any dates so far.  That is because it's tracked in source control, which means we can pull all the relevant information from the commit history.  For example, a full list of changes to any ADR could be fetched from Git with this command:

```bash
git log --format='%ci %s' -- docs/arch/
```

Likewise, when you're running your build process, you could extract the commit history which effects a single ADR:

```bash
git log --reverse --format='%ci %s' -- docs/arch/serialization-format.md
```

And then take that list and insert it into the rendered output so people can see what changed, and when:

```html
<div style="float: right">
<h2>History</h2>
    <ul>
        <li><strong>2018-09-26</strong> start serialization format docs</li>
        <li><strong>2018-09-26</strong> consider json</li>
        <li><strong>2018-09-26</strong> consider avro, inbuilt binary and custom binary</li>
        <li><strong>2018-09-27</strong> should consider thrift too</li>
        <li><strong>2018-09-28</strong> select Avro</li>
        <li><strong>2018-09-28</strong> accepted :)</li>
        <li><strong>2019-03-12</strong> accept api transport mechanisms</li>
    </ul>
</div>
```

Note how that last log entry is the deprecation of this ADR.  You can, of course, expand your log parsing only to detect Status changes etc.

## End

Hopefully, this gives you a taste of how easily useful documentation can be written, read and found.  I'm interested to hear anyone else's thoughts on whether they find this useful, or any other alternatives.
