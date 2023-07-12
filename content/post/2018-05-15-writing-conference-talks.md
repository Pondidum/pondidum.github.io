+++
date = '2018-05-15T00:00:00Z'
tags = ['productivity', 'talks', 'writing']
title = 'Writing Conference Talks'

+++

I saw an interesting question on twitter today:


> Hey, people who talk at things: How long does it take you to put a new talk together?
>
> I need like 50 hours over at least a couple of months to make something I don't hate. I'm trying to get that down (maybe by not doing pictures?) but wondering what's normal for everyone else.

[Source](https://twitter.com/whereistanya/status/995653828933496832)

I don't know how long it takes me to write a talk - as it is usually spread over many weeks/months, worked on as and when I have inspiration.  The actual processes is something like this:


1. Think it through

    The start of this is usually with an idea for a subject I like a lot, such as Strong Typing, Feature Toggles, or Trunk Based Development.  Where I live I walk everywhere (around 15k to 20k steps per day), which gives me a lot of time to think about things.

2. Giant markdown file of bullet points which I might want to cover

    I write down all the points that I want to talk about into one markdown file, which I add to over time.  I use the github checkbox markdown format (`* [ ] some point or other`) so I can tick thinks off later.

3. Rough order of points at the bottom

    At the bottom of this notes file, I start writing an order of things, just to get a sense of flow.  Once this order gets comfortable enough, I stop updating it and start using the real slides file.

4. Start slides writing sections as I feel like it

    I start with the title slide, and finding a suitable large image for it.  This takes way longer than you might imagine!  For the rest of the slides, I use a mix of titles, text and hand drawn images.

    I use OneNote and Gimp to do the hand drawn parts, and usually the [Google Cloud Platform Icons](https://cloud.google.com/icons/), as they're the best looking (sorry Amazon!)

    Attribute all the images as you go.  Much easier than trying to do it later.

4. Re-order it all!

    I talk bits of the presentation through in my head, and shuffle bits around as I see fit.  This happens a lot as I write the slides.

5. Talk it through to a wall

    My wall gets me talking to it a lot.  I talk it through outloud, and make note of stumbling points, and how long the talk takes, adding speaker notes if needed.

6. Tweaks and re-ordering

    I usually end up making last minute tweaks and order switches as I figure out how to make something flow better.  I am still not happy with some transitions in my best talks yet!

I write all my talks using [RevealJS](https://github.com/hakimel/reveal.js), mostly because I can write my slides as a markdown file and have it rendered in the browser, and partly because I've always used it.

To get things like the Speaker Notes view working, you need to be running from a webserver (rather than just from an html file on your filesystem.)  For this I use [NWS](https://www.npmjs.com/package/nws), which is a static webserver for your current working directory (e.g. `cd /d/dev/presentations && nws`).

Currently, I am trying to work out if I can use Jekyll or Hugo to generate the repository for me, as all the presentations have the same content, other than images, slides file, and a customise.css file.  Still not sure on how best to achieve what I am after though.

You can see the source for all my talks in my [Presentations Repository](https://github.com/pondidum/presentations) on github.  The actual slides can be seen on my website [here](https://andydote.co.uk/presentations/), and the videos (if available), I link to [here](https://andydote.co.uk/talks/).

