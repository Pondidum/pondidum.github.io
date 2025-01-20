+++
title = 'Explicit vs Implicit code'
tags = [ "golang", "debugging", "typescript" ]
+++

A system I am working on at the moment started giving errors occasionally, say 5 times out of 10,000 messages or so.  The error was pretty straightforward:

```
json: cannot unmarshal array into Go struct field Thing.Parts of type Parts
```

The data structure it is referring to looks like this:

```go
type Thing struct {
	Parts Parts
}

type Parts struct {
	Part []Part
}
```

Which represents the (slightly weird) json structure we receive in a message:

```json
{
	"thing": {
		"parts": {
			"part": [
				{ "name": "one" },
				{ "name": "two" }
			]
		}
	}
}
```

However, rarely, we receive a json document which looks like this instead, where the `parts` struct has instead become an array, with one object containing the `part` array:

```json
{
	"thing": {
		"parts": [	
			{
				"part": [
					{ "name": "one" },
					{ "name": "two" }
				]
			}
		]
	}
}
```

In the interest of keeping the software running while digging through for the root cause of this, I added an implementation of the `json.Unmarshaler` interface to the `Parts` struct to allow it to handle both forms of json:

```go
// duplicate of the Parts type, to prevent recursive calls to the UnmarshalJSON method
type dto struct {
	Part []Part
}

func (i *Parts) UnmarshalJSON(b []byte) error {

	// this is the standard format that json arrives in.
	normal := dto{}

	err := json.Unmarshal(b, &normal)
	if err == nil {
		t.Part = normal.Part
		return nil
	}

	// sometimes, we get json with an extra array, so if we get an error about that,
	// try the alternative structure
	if jsonErr, ok := err.(*json.UnmarshalTypeError); ok && jsonErr.Value == "array" {

		weird := []dto{}
		if err := json.Unmarshal(b, &weird); err != nil {
			return err
		}

		if len(weird) > 0 {
			t.Part = weird[0].Part
		}
		return nil
	}

	return err
}
```

When I opened a pullrequest about this, one of my colleagues approved it, but also noted:

> for once typescript would solve something more cleanly in my opinion

And I agree, after deserialising, doing something like this is much less code, and basically has the same result.

```ts
const thing = JSON.parse(message);

if (Array.isArray(thing.parts)) {
	thing.parts = thing.parts[0]
}
```

## Down the Rabbit Hole

Tracing back through the system to figure out where the message came from lead me back to a system which parses an XML document and, after doing some work on the result, emits the json message we handle.  The XML itself has a pretty reasonable structure (and far larger than I am showing here, with tens, if not hundreds of nodes):

```xml
<Thing>
	<Parts>
		<Part name="one" />
		<Part name="two" />
	</Parts>
</Thing>
```

Which it mangles into that weird json structure.  It does, however, do some sanitisation to the `Thing` before writing it out, and I found one for dealing with the `Parts` property:

```ts
// if there is only one part, the parser doesn't emit an array, so force an array.
if (!Array.isArray(thing.parts.part)) {
	thing.parts.part = [ part ]
}
```

Interesting!  but this is a different bug to the one we've just seen; in our case the `Parts` became an array...

Checking the original XML file which was processed, it looked entirely normal until I noticed that it has two `Parts` nodes:

```xml
<Thing>
	<Parts>
		<Part name="one" />
		<Part name="two" />
	</Parts>
	<!-- many nodes later -->
	<Parts>
		<Part name="three" />
		<Part name="four" />
	</Parts>
</Thing>
```

So the fix is to add another sanitisation to our parser:

```ts
if (Array.isArray(thing.parts)) {
	thing.parts = {
		part = flatMap(thing.parts)
	}
}
```

This fixes the data as soon as it appears in our system; however, searching our codebase revealed that, up until this fix, many places had been missing data, or incorrectly handling the data.

While the TypeScript types written for the `Thing` are correct, that doesn't help when the data is supplied at runtime and apparently can have varying shapes.

## The Tradeoff

The tradeoff between typescript/javascript and Go feels like this to me:

Go causes me to notice when something isn't working, as errors start being returned about data not matching the shape it was expected to be in.  Fixing the issues in general, require more code than the same fix in typescript would.

Typescript has short code, but as its only a compile-time type checking system, when weird data starts arriving, you don't get any errors (directly, things later on can break however.)

For me, I would rather have slightly longer code which is more explicit, and tells me when something goes wrong, rather than silently continuing.  The likelihood of a silent error in serialisation leading to data loss or corruption is just too high.
