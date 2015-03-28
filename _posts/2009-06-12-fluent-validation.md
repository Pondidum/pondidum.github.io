---
layout: post
title: Fluent Validation
tags: design code net
permalink: fluent-validation
---

A few days a go i was going through my bookmarks, and came accross [this post][fluent-validation] on the GetPaint.Net blog about using a fluent interface for parameter validation.

After reading the article, I tried the code out at home, and was very impressed.  Not only does it read well, but also does not create any objects untill a piece of validation fails.  Very nice.

However i wanted to use this at work, and this presented me with a problem.  Work only has VS2005, which means no extension methods, which are the crux of how this validation method works.

I spent a while trying to see if it was possible to keep the fluent interface and not instantiate any objects until something fails.  In the end i settled for this method which only creates one object.

{% highlight vbnet %}
Public Class Validate

    Public Shared Function Begin() As ValidationExpression
        Return New ValidationExpression
    End Function

    Public Class ValidationExpression

        Private _validation As Validation = Nothing

        Friend Sub New()
        End Sub

        Public Function IsNotNull(Of T)(ByVal obj As T, ByVal name As String) As ValidationExpression
            If obj Is Nothing Then
                Init()
                _validation.AddException(New ArgumentNullException(name))
            End If

            Return Me
        End Function

        Public Function IsPositive(ByVal value As Integer, ByVal name As String) As ValidationExpression

            If value < 0 Then
                Init()
                _validation.AddException(New ArgumentOutOfRangeException(name, "must be positive, but was " & value.ToString))
            End If

            Return Me
        End Function

        Public Function Check() As ValidationExpression

            If _validation Is Nothing Then
                Return Me
            End If

            If _validation.Exceptions.count = 1 Then
                Throw New ValidationException(_validation.Exceptions(0))
            Else
                Throw New ValidationException(New MultiException(_validation.Exceptions))
            End If

        End Function

        Private Sub Init()
            If _validation Is Nothing Then
                _validation = New Validation
            End If
        End Sub

    End Class
End Class
{% endhighlight %}

The rest of the code used is identical to Rick Brewster's Article, so [head over there][fluent-validation] to see it in all its (well written) glory.

[fluent-validation]: http://blog.getpaint.net/2008/12/06/a-fluent-approach-to-c-parameter-validation/
