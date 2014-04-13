---
layout: post
title: Creating a FubuMvc website
tags: code, net
permalink: creating-a-fubumvc-website
---

* Add new Empty Web Application to your solution
* PM> Install-package fubumvc
* Add folder Features
* Add folder Features\Home
* Add Features\Home\HomeInputModel.cs
* Add Features\Home\HomeViewModel.cs
* Add Features\Home\HomeEndpoint.cs
* Add Features\Home\Home.spark
* Setup application (ConfigureFubuMVC.cs)
{% highlight c# %}
Actions.FindBy(x =>
{
	x.Applies.ToThisAssembly();
	x.IncludeClassesSuffixedWithEndpoint();
});

Routes.HomeIs<HomeInputModel>();

Routes.ConstrainToHttpMethod(x => x.Method.Name.Equals("Get", StringComparison.OrdinalIgnoreCase), "GET");
Routes.IgnoreControllerNamespaceEntirely();	//removes /features/home/ from the start of urls
Routes.IgnoreMethodSuffix("Get");		//removes the trailing /get from our urls
{% endhighlight %}

*  HomeViewModel.cs:
{% highlight c# %}
public String Message { get; set; }
{% endhighlight %}

* HomeEndpoint.cs:
{% highlight c# %}
public HomeViewModel Get(HomeInputModel input)
{
	return new HomeViewModel { Message = "Dave" };
}
{% endhighlight %}

* Home.spark
{% highlight html %}
<viewdata model = "Dashboard.Features.Home.HomeViewModel" />
<h1>Hello ${Model.Message}</h1>
{% endhighlight %}

* Add folder Features\Test
* Add Features\Test\TestInputModel.cs
* Add Features\Test\TestViewModel.cs
* Add Features\Test\TestEndpoint.cs
* Add Features\Test\Test.spark
* TestEndpoint.cs:
{% highlight c# %}
public TestViewModel Get(TestInputModel input)
{
	return new TestViewModel();
}
{% endhighlight %}

* Test.spark:
{% highlight html %}
<viewdata model = "Dashboard.Features.Test.TestViewModel" />
<h1>Hello ${Model.Message}</h1>
{% endhighlight %}

* Home.spark:
{% highlight c# %}
!{this.LinkTo<TestInputModel>().Text("Test")}
{% endhighlight %}
