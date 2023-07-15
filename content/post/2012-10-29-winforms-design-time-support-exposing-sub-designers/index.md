+++
date = '2012-10-29T00:00:00Z'
tags = ['design', 'controls']
title = 'Winforms Design Time support: exposing sub designers'

+++

When writing a [UserControl][1], it is often desired to expose one or more of the sub-controls design-time support to the user of your control.  It is reasonably straight forward to do, and here is a rundown of how:

We start off with our UserControl, in this case the imaginatively named `TestControl`:

![The TestControl][2]

The code behind looks like this:

```csharp
[Designer(typeof(TestControlDesigner))]
public partial class TestControl : UserControl
{
	public TestControl()
	{
		InitializeComponent();
	}

	[DesignerSerializationVisibility(DesignerSerializationVisibility.Content)]
	public ToolStrip ToolStrip
	{
		get { return tsMain; }
	}

}
```

The first attribute on the class (`[Designer(typeof(TestControlDesigner))]`) just instructs that we want it to use our own custom designer file (which we create in a minute).
The next important point is the addition of the `ToolStrip` property, and the `DesignerSerializationVisibility` attribute that goes with it.  This informs the winforms designer that any changes made to the ToolStrip should be stored in the hosting container's designer file.  Without this attribute, no changes made in the designer would persist when you closed the designer.

Next, we add a reference to `System.Design` in the project, and create our `TestControlDesigner` class, inheriting from [ControlDesigner][3]:

```csharp
public class TestControlDesigner : ControlDesigner
{
	public override void Initialize(System.ComponentModel.IComponent component)
	{
		base.Initialize(component);

		var control = (TestControl) component;

		EnableDesignMode(control.ToolStrip, "ToolStrip");
	}
}
```

As you can see, we have very little in here.  The `Initialize` method is overriden, and we call `EnableDesignMode` on our ToolStrip (the property added to the TestControl earlier).

After compiling, we can go to our form (again, imaginatively named Form1), and add a couple of instances of `TestControl` to it from the tool box:

![The TestControl][4]

As you can see, the two control's ToolStrips contents is unique, and we have the ToolStrip's designer exposed in the forms designer.


[1]: http://msdn.microsoft.com/en-us/library/system.windows.forms.usercontrol.aspx
[2]: sub-designer-control.png
[3]: http://msdn.microsoft.com/en-us/library/system.windows.forms.design.controldesigner.aspx
[4]: sub-designer-designtime.png