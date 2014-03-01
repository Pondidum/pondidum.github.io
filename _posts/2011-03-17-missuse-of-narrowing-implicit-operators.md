---
layout: post
title: (Miss)Use of Narrowing-Implicit Operators
Tags: design, code, net
permalink: missuse-of-narrowing-implicit-operators
---

I have covered a use of Narrowing/Implicit Operators before, but I was thinking the other day about use of Fluent Interfaces, and if it was possible to have one on a cache/repository type class, that would allow you to chain options together, but stop at any point and have the result.

I gave it a go, and came up with this:

    public class Person
    {
        public string Name { get; set; }
        public int Age { get; set; }

        public Person(string name, int age)
        {
            this.Name = name;
            this.Age = age;
        }
    }

    public class PersonManager
    {
        public static PersonOptions GetPerson()
        {
            return new PersonOptions(new Person("dave", 21));
        }
    }

    public class PersonOptions
    {
        public Person Person { get; private set; }

        public PersonOptions(Person person)
        {
            this.Person = person;
        }

        public PersonOptions WaitForFreshResults()
        {
            //...
            return this;
        }

        public static implicit operator Person(PersonOptions options)
        {
            return options.Person;
        }
    }
	
Which can be used like so:

    Person p1 = PersonManager.GetPerson();
    Person p2 = PersonManager.GetPerson().WaitForFreshResults();

Which is all very well and good - but nowadays, everyone (well nearly everyone) loves the `var` keyword, so what happens if it is used like this:

    var p3 = PersonManager.GetPerson();
    var p4 = PersonManager.GetPerson().WaitForFreshResults();

Uh oh.  Thatâ€™s not a person you have in that variable, itâ€™s a PersonOptions.  The compiler does help with this, as none of your `Person` methods will be present, and the PersonOptions class does provide a Person object as a Read Only Property, so the code can be modified to use that:

    var p5 = PersonManager.GetPerson().WaitForFreshResults().Person;

I'm not entirely comfortable with using implicit conversions like this, especially with `var`, but it does work rather well, as long as you are careful.