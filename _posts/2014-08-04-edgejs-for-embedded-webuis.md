---
layout: post
title: Edge.js for Embedded Webuis
tags: design code net typing sql database orm

---

We work we have a number of windows services which each have a lot of stats they could expose.  Currently they are only interrogatable by the logfiles and from any notifications we receive.

I have been toying with the idea of hosting a website in-process which would give a simple dashboard ui and access to a live view of the log file.  The idea first struck me when I was experimenting with FubuMvc, as they have an `EmbeddedFubuMvcServer`, which is very easy to use:

{% highlight c# %}
FubuMvcPackageFacility.PhysicalRootPath = @"Backend\";

using (var server = EmbeddedFubuMvcServer.For<EmbeddedBackend>(FubuMvcPackageFacility.PhysicalRootPath))
{

    Console.WriteLine("Some long running process, with a web-backend on :5500");

    var p = server.Services.GetInstance<IProcessor>();

    var t = new Task(p.Start);
    t.Start();

    Console.ReadKey();
}
{% endhighlight %}

But while I like this, FubuMvc embedded seems like overkill.

Wouldn't it be nice if we could host an `expressjs` app inside our process?  They are very lightweight, and to get one setup is almost no coding (especially if you use the express commandline tool).

##Enter Edgejs

The [Edge.js][github-edge] project provides an in-process bridge between the .net and nodejs worlds, and allows for communication between the two...

Steps:

*	Create a new application (eg: ServiceWithEdge)

*	Create a subdirectory for the webui in your applications root (eg, next to the csproj file)
	*	ServiceWithEdge\ServiceWithEdge\webui

*	If you don't have express-generator installed, get it:
	*	`npm install -g express-generator`

*	Cd to your webui directory, and create an express application:
	*	`express` - there are some options if you want, see [the guide][express-generator-guide]

*	In visual studio, include the webui directory
	*	Mark all files as `content` and `copy if newer`

*	Add a new js file in your webui root:

{% highlight c# %}
var options;

exports.set = function (m) {
    options = m;
};

exports.getModel = function (modelName, action) {

    options.getModel(modelName, function (error, result) {

        if (error) throw error;

        action(result);
    });

};
{% endhighlight %}

*	add the edgejs package:
	*	`PM> install-package edge.js`

*	The following function will run the webui, and inject a callback for getting models from .net

{% highlight c# %}
private static void RunWebui(ModelStore store)
{
	var func = Edge.Func(@"
		var app = require('../webui/app');
		var com = require('../webui/communicator');

		app.set('port', process.env.PORT || 3000);

		var server = app.listen(app.get('port'));

		return function(options, callback) {
			com.set(options);
		};
	");

	var getModel = (Func<object, Task<object>>)(async (message) =>
	{
		return store.GetModel((string)message);
	});


	Task.Run(() => func(new
	{
		getModel
	}));
}
{% endhighlight %}

*	The last step to getting this to work is running `npm install` in the webui directory **of the build output folder**.  I use a rake file to build everything, so its just an extra task (see the entire Rakefile [here][demo-rakefile]):

{% highlight ruby %}
task :npm do |t|

	Dir.chdir "#{project_name}/bin/debug/webui" do
		system 'npm', 'install'
	end

end
{% endhighlight %}

	ny route needing data from .net just needs to require the communicator file and call `getModel`:

{% highlight js %}
var com = require('../communicator');

router.get('/', function (req, res) {

    com.getModel("index", function(value) {

        res.render('index', {
            title: 'Express',
            result: value.Iterations
        });

    });

});
{% endhighlight %}

All the code is [available on github][demo-project].

##How I am aiming to use it

I am planning on constructing a nuget package to do all of this, so that all a developer needs to do is add the package, and configure which statistics they wish to show up on the web ui.

[github-edge]: http://tjanczuk.github.io/edge/
[express-generator-guide]: http://expressjs.com/guide.html#executable
[demo-rakefile]: https://github.com/Pondidum/ServiceWithEdge/blob/master/Rakefile
[demo-project]: https://github.com/Pondidum/ServiceWithEdge
