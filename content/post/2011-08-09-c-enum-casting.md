+++
date = '2011-08-09T00:00:00Z'
tags = ['c#']
title = 'c# Enum casting'

+++

I am all for strong typing, and explicit casts, but some things in C# do seem to be a bit over-wordy.  For instance, I would quite often have code that looks like the following in VB.Net:

```vb
Public Enum Columns
	Name
	Value
	Action
End Enum

Private Sub InitialiseGrid(ByVal grid as SourceGrid.Grid)

	grid.ColumnCount = [Enum].GetValues(GetType(Columns)).Count

	grid.Columns(Columns.Name).AutoSizeMode = SourceGrid.AutoSizeMode.EnableAutoSizeView
	grid.Columns(Columns.Value).AutoSizeMode = SourceGrid.AutoSizeMode.EnableAutoSizeView | SourceGrid.AutoSizeMode.EnableStretch

	grid.Columns(Columns.Action).AutoSizeMode = SourceGrid.AutoSizeMode.None
	grid.Columns(Columns.Action).Width = 30

	'etc...

End Sub
```

The problem arrives when you try to write the same in C#, specifically the part when accessing the Columns collection using the enum:

```csharp
grid.Columns[Columns.Name].AutoSizeMode = SourceGrid.AutoSizeMode.EnableAutoSizeView;
```

Sorry, no dice, you must cast the enum to an int first.  What? Really? It's an int value at heart anyway (by default at any rate) and you can even specify an Enum to use an Int (or other numeric data type) if you should so wish, so why does this need an explicit cast?  This just looks nasty, in my opinion:

```csharp
grid.Columns[(int)Columns.Name].AutoSizeMode = SourceGrid.AutoSizeMode.EnableAutoSizeView;
```

I can only think of two ways of maintaining the cleanness that the VB provides, and both are more effort.  The first is to create an ExtensionMethod for the Grid with the following signature, doing the casting inside the method, and using type inference to allow the enum to be passed straight in:

```csharp
public static ColumnInfo ColumnAt<T>(self grid Grid, T index) where T : struct
```

The second method is to not use an enum to store our column indexes, but to use a class, with constants.  The only down side I can see to this is the lack of being able to count the number of columns, based on the members (without resorting to reflection, or a lambda for finding the Max value):

```csharp
private static class Columns
{
	public const int Name = 0;
	public const int Value = 1;
	public const int Action = 2;
}
```

I am not a fan of the ExtensionMethod method, and I would use the class personally - usually hard coding the number of columns is fine, but I still prefer the concise and simple version that VB.Net allows you.
