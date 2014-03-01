---
layout: post
title: Coming From Something as opposed to Going To Something
Tags: design, code, net
permalink: coming-from-something-as-opposed-to-going-to-something
---

Over the last week I have noticed myself preferring methods being called IntegerFromString rather than StringToInteger.  Is sometimes takes me a little longer to read (only a few milliseconds, mind) but I think I am getting more used to it, and I do think it enhances readability.

The main point for readability comes from the fact that I work a lot (in my spare time when coding) on graphics processing in GDI.  When working with several different sets of coordinates it can get confusing, especially when converting between them, or having to use two different coordinate styles at once.

For instance in my current project, I deal a lot with rotation, so I am using [Polar Coordinate System][1] which specifies an angle and a length.  However as a windows form uses the Raster Coordinate System/Offset [Cartesian][2] (e.g. 0, 0 is in the Top Left), I end up converting from Polar to Cartesian to Raster.

When I was writing the functions to do this for me, I ended up naming them things like `Point F RasterFromCartesian(PointF pt);` which helped a lot as when used in code I end up with something like this:

    PointF locationRaster = RasterFromCartesian(CartesianFromPolar(angle, length));

Which keeps the keywords close together and may not seem like a huge advantage with the declaration line, but when later on in the code you see this:

    locationRaster = CartesianDistance(currentCartesian, destinationCartesian);

You can see instantly that something is wrong, as the code is assigning a Cartesian straight to a Raster variable.  By having the word Raster on the end of my variable name and the resultant type on the beginning of my function, it is very easy to see what is happening at a glance.

I admit this is probably not the best explanation; Joel Spolsky has a very good article on the subject [Here][3].

[1]: http://en.wikipedia.org/wiki/Polar_coordinate_system
[2]: http://en.wikipedia.org/wiki/Cartesian_coordinate_system 
[3]: http://www.joelonsoftware.com/articles/Wrong.html