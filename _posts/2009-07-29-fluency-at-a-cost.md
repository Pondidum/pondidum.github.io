---
layout: post
title: Fluency at a cost?
tags: design code net
permalink: fluency-at-a-cost
---

I like fluent interfaces.  I find them easy to read, and nice to program with.  However the more I write them the more I notice there is a cost associated with them.  It's not much of a cost, but it is there none the less.  To demonstrate say we have a class called `Animator`.  It has the following properties and methods on it:

    + Control
    + Distance
    + DistanceType
    + AnimationType
    + Direction
    + Time
    + Algorithm
    - Animate()

Now while you could just set all the properties and then call `Animate()`, a Fluent Interface makes thing nicer:

    Animate
		.Control(Button1)
		.Slide
		.Right
		.By(60)
		.Using(New ExponentialAlgorithm)
		.Start()

To make the interface more constrained, there are about 4 classes being used:

    Static Class Animate
      - AnimationExpression Control(Control con)

    Class AnimationExpression
      - DirectionExpression Slide()
      - DirectionExpression Grow()
      - DirectionExpression Shrink()

    Class DirectionExpression
      - DistanceExpression Up()
      - DistanceExpression Down()
      - DistanceExpression Left()
      - DistanceExpression Right()

    Class DistanceExpression
      - DistanceExpression Taking(int time)
      - StartExpression To(int position)
      - StartExpression By(int distance)

    Class StartExpression
      - StartExpression Using(IAlgorithm algorithm)
      - void Start()

The first class (Animation Expression) creates an instance of the `Animator` class, and then that is passed into the constructor of the other classes, after having a property set e.g.:

    DistanceExpression Up {
        _animator.DirectionType = Animator.DirectionTypes.Up
        return new DistanceExpression(_animator)
    }

So when you use the Fluent Interface, you end up with around 6 extra instances created rather than just 1 (the animator).  This might not be much of an overhead as each class is fairly small, but if you are doing a lot of animations, it is going to add up (depending on how often the GC sees fit to destroy them).

Compare this fluent interface to the one created for [parameter validation by Rick Brewster][1] that uses Extension Methods so that he creates no extra instances unless there is an error detected.

I am not entirely sure how much of an impact this would have on a program, but its definitely something worth remembering when writing fluent interfaces for your classes.


[1]: http://blog.getpaint.net/2008/12/06/a-fluent-approach-to-c-parameter-validation/
