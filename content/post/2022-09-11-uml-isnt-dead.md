+++
tags = ['uml', 'design']
title = "The reports of UML's death are greatly exaggerated"

+++

This is in response to the [recent](https://buttondown.email/hillelwayne/archive/why-uml-really-died/) [posts](https://garba.org/posts/2021/uml/) about the death of UML;  while I think some parts of UML have fallen ill, the remaining parts are still alive, and useful to this day.

## TLDR

Out of [14 types of diagram](https://creately.com/blog/diagrams/uml-diagram-types-examples/) there are 3 that I use on a regular basis: **Activity Diagram**, **State Machine Diagram**, and **Sequence Diagram**.  I think the Timing Diagram is borderline, but I can only think of a couple of occasions when it has been useful.

Writing the diagrams in text and rendering them with [Mermaid] makes including them in documentation and websites painless, and the project is under active development.

## What I use often

The diagram I use the most is the **Sequence Diagram**;  It's a great way to document how multiple systems (or micro services) will interact with each other.  This diagram type has worked really well on both physical whiteboards, and in documentation.  For example, part of a diagram I used to help design a system recently:

{{<mermaid align="left">}}
sequenceDiagram
    participant api
    participant cas
    participant storage

    api ->>+cas: dependencies

    alt exists
        cas ->>+storage: get key
        storage ->>-cas: key, date
    else doesn't exist
        cas ->>+storage: get key
        storage ->>-cas: [not found]
        cas ->>+storage: write key, now()
        storage ->>-cas: [ok]
    end

    cas ->>-api: key path
{{< /mermaid >}}

Next most used is the **Activity Diagram**, also more commonly known as a **Flow Chart**.  This is mostly used when discribing an algorithm or process without needing to indicate different participants in the algorithm.


{{<mermaid align="left">}}
graph LR
    deps[key Dependencies]
    in_store{key in Storage?}
    write_to_store[key + current date to storage]
    update_date[Update store date from file]
    return[return key path]

    deps --> in_store
    in_store -->|Yes| update_date --> return
    in_store -->|No| write_to_store --> return
{{< /mermaid >}}

The last type that I use often is the **State Machine Diagram**;  I think that State Machines themselves are an under utilised design pattern, and that a lot of complex problems can be rendered into the state pattern quite easily.

In a previous job there was a state machine with around 34 different states; being able to render this in a diagram made understanding the process much more approachable; even our support staff used the diagram to answer user questions.

For example, a line processor could be represented as follows, where depending on the kind of error the process will either terminate, or skip the line and wait for the next:

{{<mermaid align="left">}}
stateDiagram
    state "Wait for Line" as wait
    state "Process Line" as process

    [*] --> wait
    wait --> process
    process --> wait
    process --> Error
    Error --> wait
    Error --> [*]
{{< /mermaid >}}

## Who else is using UML?

The [Mermaid] library to render these kind of diagrams is being integrated all over the place:  Github uses it to handle any ` ```mermaid ` blocks in your markdown files.

Likewise, [Hugo] uses it when rendering your markdown content too (which is what has drawn these nice diagrams here.)

There are [extensions for vscode][mermaid-extension] to render them too.

## Why should I care?

Because explaining concepts is hard; and pictures often help.  The old phrase of a picture being worth a thousand words is applicable here; its far easier to glance over a diagram than read a few paragraphs of prose.

This being the case, it is a good idea to have some common formats, standards or symbols to use, making the diagrams even clearer to people who already know the symbols (and don't forget to explain them if they don't know, and perhaps include a legend in your diagram.)

When creating these diagrams was done in tools like Visio, and then screenshots embedded in documents (usually buried in a wiki where no one will find them), the barrier to making a useful diagram was high.  Being able to embed them in the markdown in your repo, _so they can be read from code even if they aren't rendered_ lowers that barrier to making useful diagrams considerably.

So do your co-workers, contributors, and your future-self a favour, and add some simple diagrams to your docs.


[mermaid]: https://mermaid-js.github.io/mermaid/#/
[hugo]: https://gohugo.io/
[mermaid-extension]: https://marketplace.visualstudio.com/items?itemName=bierner.markdown-mermaid
