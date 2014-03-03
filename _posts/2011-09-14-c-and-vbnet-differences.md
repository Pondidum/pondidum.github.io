---
layout: post
title: C# and Vb.Net Differences
Tags: code, net
permalink: c-and-vbnet-differences
---

So I have been doing some work that involves C# and VB libraries and apps using each other, and have noticed a lot of subtle differences between the two languages.

Declaration of types inside an interface:
---

{% highlight vbnet %}
Public Interface ITesting

	ReadOnly Property Test() As TestData

	Class TestData
		Public Sub New()
			StringProperty = "testing"
			IntProperty = 1234
		End Sub

		Public Property StringProperty() As String
		Public Property IntProperty() As Integer
	End Class

End Interface
{% endhighlight %}

However in C#, you cannot declare types inside an interface, however it is quite happy to consume one create in a VB project:

{% highlight c# %}
var test = new VbLib.ITesting.TestData();
{% endhighlight %}

That is not to say it is a good thing to do - I have encountered problems with XML Deserialization not working if it needed to deserialize an enum that was declared inside an interface.

Indexed Properties
---

Again, this is perfectly legal in VB:

{% highlight vbnet %}
Public Class CustomCollection
	Inherits List(Of CustomObject)

	Default Public Shadows ReadOnly Property Item(ByVal index As Integer) As CustomObject
		Get
			Return MyBase.Item(index)
		End Get
	End Property

	Public ReadOnly Property IndexedReadOnly(ByVal index As Integer) As CustomObject
		Get
			Return Me(index)
		End Get
	End Property

	Public Property IndexedReadWrite(ByVal index As Integer) As CustomObject
		Get
			Return Me(index)
		End Get
		Set(ByVal value As CustomObject)
			MyBase.Item(index) = value
		End Set
	End Property

	Public ReadOnly Property EnumIndexed(ByVal type As CustomObject.CustomTypes) As CustomObject
		Get
			Return Me.FirstOrDefault(Function(x) x.Type = type)
		End Get
	End Property

End Class
{% endhighlight %}

It compiles, and runs fine from VB:

{% highlight vbnet %}
Public Sub Test()
	Dim collection = New CustomCollection()
	Dim output = collection.EnumIndexed(CustomObject.CustomTypes.Testing)
End Sub
{% endhighlight %}

However trying to consume this from C# will not work:

{% highlight c# %}
var item = collection.EnumIndexed(VbLib.CustomObject.CustomTypes.Other);
{% endhighlight %}

But like this will:

{% highlight c# %}
var item = collection.get_EnumIndexed(VbLib.CustomObject.CustomTypes.Other);
{% endhighlight %}
