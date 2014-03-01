---
layout: post
title: Creating a FubuMvc website
Tags: code, net
permalink: creating-a-fubumvc-website
---

* Add new Empty Web Application to your solution
* PM> Install-package fubumvc
* Add folder Features
* Add folder Features\Home
* Add Features\Home\HomeInputModel.cs
* Add Features\Home\HomeViewModel.cs
* Add Features\Home\HomeController.cs
* Add Features\Home\Home.spark
* Setup application (ConfigureFubuMVC.cs)

		Actions.FindBy(x =>
		{
			x.Applies.ToThisAssembly();
			x.IncludeClassesSuffixedWithController();
		});

		Routes.HomeIs<HomeInputModel>();

		Routes.ConstrainToHttpMethod(x => x.Method.Name.Equals("Get", StringComparison.OrdinalIgnoreCase), "GET");
		Routes.IgnoreControllerNamespaceEntirely();	//removes /features/home/ from the start of urls
		Routes.IgnoreMethodSuffix("Get");		//removes the trailing /get from our urls

*  HomeViewModel.cs:

		public String Message { get; set; }

* HomeController.cs:

		public HomeViewModel Get(HomeInputModel input)
		{
			return new HomeViewModel { Message = "Dave" };
		}

* Home.spark
	
		<viewdata model = "Dashboard.Features.Home.HomeViewModel" />
		<h1>Hello ${Model.Message}</h1>

* Add folder Features\Test
* Add Features\Test\TestInputModel.cs
* Add Features\Test\TestViewModel.cs
* Add Features\Test\TestController.cs
* Add Features\Test\Test.spark
* TestController.cs:
	
		public TestViewModel Get(TestInputModel input)
		{
			return new TestViewModel();
		}

* Test.spark:

		<viewdata model = "Dashboard.Features.Test.TestViewModel" />
		<h1>Hello ${Model.Message}</h1>

* Home.spark:

		!{this.LinkTo<TestInputModel>().Text("Test")}
