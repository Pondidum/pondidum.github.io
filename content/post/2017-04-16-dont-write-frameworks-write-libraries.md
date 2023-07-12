+++
date = '2017-04-16T00:00:00Z'
tags = ['architecture', 'code']
title = "Don't write Frameworks, write Libraries"

+++

Programmers have a fascination with writing frameworks for some reason.  There are many problems with writing frameworks:

### Opinions
Frameworks are opinionated, and will follow their author's opinions on how things should be done, such as application structure, configuration, and methodology.  The problem this gives is that not everyone will agree with the author, or their framework's opinions.  Even if they really like part of how the framework works, they might not like another part, or might not be able to rewrite their application to take advantage of the framework.

### Configurability
The level of configuration available in a framework is almost never correct.  Not only is there either too little or too much configuration options, but how the configuration is done can cause issues.  Some developers love conventions, other prefer explicit configuration.

### Development
Frameworks suffer from the danger of not solving the right problem, or missing the problem due to how long it took to implement the framework.  This is compounded by *when* a framework is decided to be developed, which is often way before the general case is even recognised.  Writing a framework before writing your project is almost certain to end up with a framework which either isn't suitable for the project, or isn't suitable for any other projects.

## What about a library or two?
If you want a higher chance at success, reduce your scope and write a library.

A library is usually a small unit of functionality, and does one thing and does it well (sound like microservices or Bounded Contexts much?).  This gives it a higher chance of success, as the opinions of the library are going to effect smaller portions of peoples applications.  It won't dictate their entire app structure.  They can opt in to using the libraries they like, rather than all the baggage which comes with a framework.

## But I really want to write a framework

Resist, if you can!  Perhaps a framework will evolve from your software, perhaps not.  What I have found to be a better path is to create libraries which work on their own, but also work well with each other.  This can make this more difficult, but it also give you the ability to release libraries as they are completed, rather than waiting for an entire framework to be "done".

## Some examples

These are some libraries I have written which solve small problems in an isolated manner

* [Stronk](https://github.com/pondidum/stronk) - A library to populate strong typed configuration objects.
* [FileSystem](https://github.com/pondidum/FileSystem) - Provides a FileSystem abstraction,  with decorators and an InMemory FileSystem implementation.
* [Finite](https://github.com/pondidum/Finite) - a Finite State Machine library.
* [Conifer](https://github.com/pondidum/conifer) - Strong typed, Convention based routing for WebAPI, also with route lookup abilities

So why not write some libraries?
