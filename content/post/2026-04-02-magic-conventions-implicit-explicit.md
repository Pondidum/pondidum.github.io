+++
title = 'On Magic, Convetions, and Implicit vs Explicit'
tags = [ "architecture", "design", "developer experience", "debugging" ]
+++

Earlier in my career I was working as a C# developer, and had read a lot about testing and testability from the Alt.Net crowd, a lose group of people with ideas outside of how Microsoft did things in C#.  When I came to needing to write an API, the impossibility of testing in ASP.NET made me look elsewhere.

ASP.NET made heavy usage of Annotations, where you add decorators to methods which at runtime are interpreted by a framework to add functionality, or configure how a function is used.  It wasn't uncommon to see a method with 5 or more Annotations:

```c#
HttpRoute("/user")
public class UserAddressesController : Controller {

  HttpPost()
  HttpPut()
  HttpRoute("/addresses")
  Authenticate()
  Accept("application/json")
  Response("application/json")
  public function AddAddress() {
    var body = HttpContext.Body;
    //...
  }
}
```

Other tools in the .NET world used the concept of Marker interfaces to remove some of the inheritance.  A marker interface doesn't have any methods, but by implementing it the framework finds the class at runtime to use.

FubuMVC ("For Us, By Us") was heavily inspired by Ruby, and more specifically Rails.  It favoured **Convention over Configuration** heavily.  Rather than Annotations or Marker Interfaces, it used naming conventions.  Name your class `ThingController`? it became a controller.  Name it `OtherEndpoint` and it became an Endpoint.  I don't currently remember what the difference was.  Name your method `GetUserAddress` and it would handle `GET` requests to `/user/address`

This was fine when you were doing the obvious parts of the application, like Controllers, and Views and Models etc, but when you started to need to do other things, it got a little harder.  How do you do some error handling decoration to all routes? Add a class called `ErrorFilter`, with a method with the right signature, and it just works.  As the [author wrote in a retrospective](https://groups.google.com/g/fubumvc-devel/c/FWhrZcLpAso/m/VuFu8GpyfyYJ?pli=1) about the project, the documentation was...limited to put it lightly.  A lot of questions and answers were in a chatroom on Gitter (I think) which was hard to search.  

This experience wasn't all negative.  Far from it in fact; it had amazing testability, worked with a decent dependency injection container, and didn't use Annotations at all, which was fantastic.

The system worked well once you knew how it worked, but new developers on the system?  They had better hope there was a more experienced developer around to show them how to add routes, handlers, controllers, and whatnot.

Over time, I have drifted further and further away from Convention over Configuration in this sense.  Conventions in a codebase are still important; things like "we name all identifiers `ThingId`", or "cli actions are referred to as Commands" give a lot of consistency, without an extra mental hurdle.

## Being the New Guy

At work recently, my friend and I have been given a service to take ownership of, modernise, and start implementing new features in.  The codebase could be generously described as "awaiting care", but, it does work.  The issue we are having however is that it is heavily Convention over Configuration based, highly abstracted, and very un-explicit everywhere.  It is borderline magical how the system actually works.

The project is Java based which in itself is not a bad thing, but it leans heavily on Annotations (or Attributes), and two gigantic "common" libraries which both can do so much based solely off of configuration values.

For example, the codebase uses some kind of role based authentication via an Annotation on the class, and then further annotations on the methods.  I want to know what is actually doing the authentication, but in the `.properties` files there are several oauth clients configured, all with fairly generic names like "oauth", "clients", "accounts",  kind of names.  None of them are referenced in application code, and it took a lot of digging through shared libraries to figure out what was going on.

Another example are the many HTTP route handlers, which are also configured with Annotations, and `@bean` magic.  I want to find the handler for some long path, ending with `/bulk`, which is quite hard to do when the path is split across multiple Annotations (class level, method level), and half the time (yay for inconsistencies!) the route is defined on an abstract class then I have to find the (only) `ClassWhateverImpl` to see the handler.

These kind of things are problematic for several reasons:

1. I am owning the service.  If auth breaks, I should be able to debug it
2. We want to refactor and remove unused dependencies.  Are these properties used?
3. We might replace the service, can we rely on a common auth module for another language?

## What I would do instead

Be explicit.  Figure out how the team wants to define routes, then do that one thing everywhere.  I love opening a `server.go` file and finding a method that looks like this:

```go
func NewHttpAPI(...) {
  server, err := createServer(...)
  // imagine error handling here

  signals := withOsSignals(ctx)

  withGracefulShutdown(ctx, server, signals)
  withTracing(ctx, server)
  withMetrics(ctx, server)
  withPanicRecovery(ctx, server)

  server.Handle("GET /_info", infoHandler(config, deploymentInfo))
  server.Handle("GET /_info/ready", readyHandler(ctx, db, cache, signals))
  server.Handle("GET /_info/live", liveHandler(ctx))

  auth, err := authMiddleware(db)

  server.Handle("GET /users", getUsersHandler(cache))
  server.Handle("POST /users", auth("user:create"), creatUserHandler(db, cache))
  server.Handler("DELETE /users/{userid}", auth("user:delete"), deleteUserHandler(db, cache))
  //...
}
```

This code tells me so many things all in once place:

- all routes are handled by functions called `...Handler`
- the server is listening to some OS signals
- the signals are used for graceful shutdown
- there are tracing, and metrics globally
- there is a panic handler to prevent crashes
- there is a db and a cache
- listing users comes from cache, NOT db
- listing users is anonymous allowed
- creating a user requires `user:create` permission, deleting requires `user:delete`

This code is possibly longer than a convention based system, but typing speed is not the part of development that slows people down (thinking is).  Code is read far more than written, and having all this information explicitly defined means any new person to the project has a reasonable chance of getting started.

There are also good abstractions hinted at:  I don't need to know how graceful shutdown works, or how OS signals are handled.  If graceful shutdown stops working, the first place I am checking is `withGracefulShutdown`, followed by `withOsSignals`.

## For the future

Be explicit, reduce magic, and remember code is read more than written.

Be kind to the new people joining your codebase.  It might just be you in the future!
