---
layout: post
title: "Model View Presenters: Introduction"
Tags: design, net
permalink: model-view-presenter-introduction
---

Table of Contents
----------------------------------

* **Introduction**
* [Presenter to View Communication][6]
* [View to Presenter Communication][7]
* [Composite Views][8]
* Presenter / Application communication
* ...

What is MVP?
------------

I first came across MVP in [Jeremy Miller's][1] [Build Your Own Cab series][2], and have been using and improving how I work with this style ever since.  Model View Presenters tend to come in one of two forms: [Passive View][3], and [Supervising Controller][4].  I am a fan of the Passive View variety, primarily for the testing aspect, but also as I find it provides me with the best level of separation.

The code ends up structured like this:

![MVP][5]

The View contains only code that enables control population and feedback.  This means the odd For loop or similar to fill a grid from a property, or feedback object construction, along the lines of `new SelectedRowData {ID = (int)row.Tag, Name = row[0].Value}`.  However, it would not contain any decision logic.

The Presenter contains code to transform Model data to something the View can display, and vice-verse.  It also contains any view logic, such as if a CheckBox is selected, then a MenuItem becomes disabled.

The Model is the data to be displayed.  This can either be an abstraction that encompasses several entities and business logic, or can be some entities themselves.




[1]: http://codebetter.com/jeremymiller/
[2]: http://codebetter.com/jeremymiller/2007/07/26/the-build-your-own-cab-series-table-of-contents/
[3]: http://martinfowler.com/eaaDev/PassiveScreen.html
[4]: http://martinfowler.com/eaaDev/SupervisingPresenter.html
[5]: /images/91.jpg
[6]: /model-view-presenters-presenter-to-view-communication
[7]: /model-view-presenters-view-to-presenter-communication
[8]: /model-view-presenters-composite-views