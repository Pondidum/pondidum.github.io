---
layout: post
title: Generating AssemblyInfo files with Gulp
tags: code c# gulp
---

When changing a project's build script over to [Gulpjs][gulp], I ran into a problem with one step - creating an `AssemblyInfo.cs` file.

My projects have their version number in the `package.json` file, and I read that at compile time, pull in some information from the build server, and write that to an `AssemblyVersion.cs` file.  This file is not tracked by git, and I don't want it showing up as a modification if you run the build script locally.

The problem is that the [gulp-dotnet-assembly-info][gulp-assembly] package doesn't support generation of files, only updating.  To get around this I used the [gulp-rename][gulp-rename] package to read a template file, and generate the non-tracked `AssemblyVersion.cs` file.

## Steps

First, create an `AssemblyVersion.base` file, and save it somewhere in your repository.  I usually put it next to the `gulpfile`, or in the projects `Properties` directory, depending on if the project has multiple assemblies or not.  This file can be added and tracked by git - it won't get changed.

```csharp
using System.Reflection;
using System.Runtime.CompilerServices;
using System.Runtime.InteropServices;

[assembly: AssemblyVersion("0.0.0")]
[assembly: AssemblyFileVersion("0.0.0")]
[assembly: AssemblyDescription("Build: 0, Commit Sha: 0")]
```

Next install the two gulp modules, and import into your `gulpfile`:

```bash
npm install gulp-rename --save
npm install gulp-dotnet-assembly-info --save
```

```javascript
var rename = require('gulp-rename');
var assemblyInfo = require('gulp-dotnet-assembly-info');
```

In the gulp file, read the `package.json` file and the environment variables.  I do this once at the begining of my `gulpfile` and use the config all over the place.

```javascript
var project = JSON.parse(fs.readFileSync("./package.json"));

var config = {
  name: project.name,
  version: project.version,
  commit: process.env.APPVEYOR_REPO_COMMIT || "0",
  buildNumber: process.env.APPVEYOR_BUILD_VERSION || "0",
}
```

Then add a task to create a new `AssemblyVersion.cs` file.  Change the `src` parameter to match where you saved the `AssemblyVersion.base` file.

```javascript
gulp.task('version', function() {
  return gulp
    .src(config.name + '/Properties/AssemblyVersion.base')
    .pipe(rename("AssemblyVersion.cs"))
    .pipe(assemblyInfo({
      version: config.version,
      fileVersion: config.version,
      description: "Build: " +  config.buildNumber + ", Sha: " + config.commit
    }))
    .pipe(gulp.dest('./' + config.name + '/Properties'));
});
```

Don't forget to reference the `AssemblyVersion.cs` file in your csproj!

You can see a full `gulpfile` with this in here: [Magistrate gulpfile][github-magistrate].

[gulp]: http://gulpjs.com/
[gulp-assembly]: https://www.npmjs.com/package/gulp-dotnet-assembly-info
[gulp-rename]: https://www.npmjs.com/package/gulp-rename
[github-magistrate]: https://github.com/Pondidum/Magistrate/blob/master/gulpfile.js
