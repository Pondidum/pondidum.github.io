---
layout: post
title: Encapsulation in Warcraft Addons - Closures
tags: design code lua warcraft
---

In the [last post][blog-addon-design] I alluded to the fact that if you put in a little leg work, you could write well encapsulated objects in lua.  There are two main ways to do this; with closures, and with metatables.  In this post we will deal with using closures, and in the next post we will cover using metatables.

## Using Closures

The simplest way to write an object in lua is with a [closure][wiki-closure]  to hide all the variables from the outside world.  For example, we can write a counter class like so:

```lua
local counter = {

	new = function()

		local count = 0

		local this = {

			increase = function()
				count = count + 1 end
			end,

			print = function()
				print("The count is " .. count .. ".")
			end,
		}

		return this

	end,
}
```

We are using a table to give us a class name, and the closure is the only method on it (called `new`).  My standard convention is to call the actual object we return `this`. The `this` object contains the public surface of our object, in this case two methods called `increase()` and `print()`.  You can use the counter like this:

```lua
local first = counter.new()

first.increase()
first.print() -- prints "The count is 1"
```

By using a closure, we limit the use of the `count` variable to only methods defined in the body of the function `new`.  This prevents anyone who uses the class from knowing how it is implemented, which is important as we are now at liberty to change the implementation without affecting our users.

A good example of this technique is in my [Dark.Combat][github-dark-combat] addon.  While writing cooldown tracking, I needed to know how many stacks of Maelstrom Weapon was the maximum, so that I could trigger a glow effect on the icon.  The problem is that the Warcraft API doesn't have a way of querying this (you can call [GetSpellCharges][wowprogramming-getspellcharges] for spells such as Conflagurate, but sadly this doesn't work on an aura.)

To solve this, rather than hard coding values into the view, or forcing the user to specify some kind of "glow at xxx stacks" parameter in the config, I wrote an object which you can be queried.  This could also be expanded later to hold additional spell data which is not available in the API.

```lua
local addon, ns = ...

local spellData = {

	new = function()

		local charges = {
			[53817] = 5,
			["Maelstrom Weapon"] = 5,

			[91342] = 5,
			["Shadow Infusion"] = 5,
		}

		setmetatable(charges, { __index = function(key) return 1 end })

		return {
			getMaxCharges = function(spellID)
				return charges[spellID]
			end,
		}

	end
}

ns.spellData = spellData.new()
```

As the implementation of `getMaxCharges` is hidden, I can change it at will - perhaps splitting my `charges` table into two separate tables, or if Blizzard kindly implemented a `GetMaxStacks(spellName)` I could call this instead and remove my `charges` table altogether.  

### Composition

We can utilise composition to create objects based off other objects, by decorating an instance with new functionality.  A slightly cut down version of the grouping code from my [Dark.Bags addon][github-dark-bags-groups] makes good use of this:

```lua
local group = {

	new = function(name, parent, options)

		local frame = CreateFrame("Frame", name, parent),
		layoutEngine.init(frame, { type = "HORIZONTAL", wrap = true, autosize = true })

		return {
			add = function(child)
				frame.add(child)
			end,
		}
	end,
}

local bag = {

	new = function(name, parent)

		local this = group.new(name, parent)

		this.populate = function(contents)

			for key, details in pairs(contents) do
				this.add(itemView.new(details))
			end

		end

		return this

	end,
}
```

Here we have two classes `group` and `bag`.  The `group` acts as our base class; it just creates a frame, and initialises a layout engine which does the heavy lifiting of laying out child frames.

In the `bag.new()` function, we create an instance of a `group` and add a `populate` method to it, and return it.  We can continue creating new classes which use `bag` and `group` as base types as we need.

### Problems with Closures

The down side to using closures is that inheritance is not really possible.  To take the `counter` example again, if you wanted to create a stepping counter, you couldn't do this:

```lua

local evenCounter = {
	new = function()

		local this = counter.new()

		this.increase = function()
			-- how do we access count?!
		end

		return this
	end
}
```

Not only can you not access the original `count` variable, but you would also have to reimplement the `print` function as it would not have access to your new counting variable.

These problems can be solved using the metatables methods in the next post, however depending on what you are doing, you could just use composition instead as outlined below.

[wiki-closure]: http://en.wikipedia.org/wiki/Closure_(computer_programming)
[blog-addon-design]: http://andydote.co.uk/2014/11/23/good-design-in-warcraft-addons.html

[github-dark-combat]: https://github.com/Pondidum/Dark.Combat
[github-dark-bags-groups]: https://github.com/Pondidum/Dark.Bags/tree/master/groups

[wowprogramming-getspellcharges]: http://wowprogramming.com/docs/api/GetSpellCharges
