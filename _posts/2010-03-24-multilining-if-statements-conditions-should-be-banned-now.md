---
layout: post
title: Multilining If statements conditions should be banned. now.
Tags: code, bug, net
permalink: multilining-if-statements-conditions-should-be-banned-now
---

Multilining if statement conditions is bad.  I was modifying some code and came across this:

{% highlight vbnet %}
If String.IsNullOrEmpty(_selectedGUID) OrElse _
_selectedGUID = FeeAgreement.GetDefaultContractAgreementGuid OrElse _
_selectedGUID = FeeAgreement.DefaultPermAgreementGuid Then

	fgFeeAgreements.SetCellCheck(rowAdded, 0, CheckEnum.Checked)
	_selectedTitle = ag.Title
	_lastIndexRowSelected = rowAdded

End If
{% endhighlight %}

Which at a glance looks like this:

> Single Line If
> Variable Assignment
> Variable Assignment

One person suggested that if someone had to do multiline the condition they could at least indent it.  That's not much good either though:

{% highlight vbnet %}
If String.IsNullOrEmpty(_selectedGUID) OrElse _
	_selectedGUID = FeeAgreement.GetDefaultContractAgreementGuid OrElse _
	_selectedGUID = FeeAgreement.DefaultPermAgreementGuid Then

	fgFeeAgreements.SetCellCheck(rowAdded, 0, CheckEnum.Checked)
	_selectedTitle = ag.Title
	_lastIndexRowSelected = rowAdded

End If
{% endhighlight %}

Looks like this:

>If Condition Then
>	Variable Assignment
>	Variable Assignment

You could one line the whole thing, which while I think is better than multi line conditionals, still isn't great as I cant see all of it on a normal sized screen (read "work supplied screen").

{% highlight vbnet %}
If String.IsNullOrEmpty(_selectedGUID) OrElse _selectedGUID = FeeAgreement.GetDefaultContractAgreementGuid OrElse _selectedGUID = FeeAgreement.DefaultPermAgreementGuid Then

	fgFeeAgreements.SetCellCheck(rowAdded, 0, CheckEnum.Checked)
	_selectedTitle = ag.Title
	_lastIndexRowSelected = rowAdded

End If
{% endhighlight %}

So, Why not just do it as suggested in Code Complete, which fits on my screen and explains the comparisons:

{% highlight vbnet %}
Dim isContract = (_selectedGUID = FeeAgreement.GetDefaultContractAgreementGuid)
Dim isPerm = (_selectedGUID = FeeAgreement.DefaultPermAgreementGuid)

If String.IsNullOrEmpty(_selectedGUID) OrElse isContract OrElse isPerm Then

	fgFeeAgreements.SetCellCheck(rowAdded, 0, CheckEnum.Checked)
	_selectedTitle = ag.Title
	_lastIndexRowSelected = rowAdded

End If
{% endhighlight %}

I don't know who wrote the above original code, and I don't much care either.
I do however think that the people who like the original style are clinically insane...and I work with at least one like this!

Some unit tests wouldn't go amiss either.  Well, tests of any kind would be a good start...
