+++
title = 'Architecture Testing'
tags = ['architecture', 'adr', 'testing']
+++

One of the many reasons given for using microservices rather than a mono repository is that it enforces boundaries between services/modules.  However, there are ways to achieve strong boundaries between modules/services in one repository, using tools which are already available: test runners.

Given a repository with the following structure:

```
.
├── libraries
│   ├── core
│   ├── events
│   └── ui
├── services
│   ├── catalogue
│   ├── billing
│   └── shipping
└── tools
    └── admin-cli
```

There are a few rules we should enforce:

- Services cannot reference each other
- tools cannot reference each other
- Services cannot reference tools
- Libraries can only reference other libraries
- Libraries cannot have circular dependencies

There are also the conventions that we want to enforce:

- Feature folders should be used, not `models`, `views` and `controllers`
- Specific libraries should not be used
- all services should expose a `/api/stats` endpoint

How to write these tests will vary greatly depending on what programming languages and tools you use, but I know for sure they can be written in Go, C#, and TypeScript.  Not only that, but the tests can be written in a different language than the applications; in this example, our applications are written in a mix of NodeJS and Go, and the architectural tests are written in Go.

## Testing for a Convention

The convention we will test for is that we strongly prefer folder-by-feature over folder-by-type.

The test itself uses a couple of helper methods: `repositoryFolders` returns a slice of every folder recursively in the project, with information such as all child folders and all child files, along with names, paths, etc., populated.

The `hasLayers` function itself is just checking if the direct children of a folder contain "models", "views" and "controllers" or "models", "views" and "presenters".


```go
func TestFolderByFeature(t *testing.T) {
  folders := repositoryFolders()

  for _, folder := range folders {
    if hasLayers(folder) {

      assert.Failf(t, "found type folders, not slices", wrap80(`
It looks like '%s' is using this folder structure, known as "folder-by-type", which is discouraged:

%s

Instead, you should use folders-by-feature:

%s

For more information, see this ADR: ./docs/arch/005-folder-layout.md

If this test failure is a false positive, please let us know, or you can either improve the test or add your folder path to the '.architecture-ignore' file.  Here is the fragment that can be added:

%s
      `, folder.Path, layers(folder), slices(folder), folderByTypeArchitectureIgnore(folder)))
    }
  }
}
```
The error message in this kind of test is very important; it needs to cover:

- What was wrong
- Where the failure was (i.e. the path)
- Why this is considered wrong (with links to more information if needed)
- How to fix it
- How to add an exception to the rules (if desired)
- How to handle false positives

For example, this is what the rendered output of the test above looks like, showing the folder that was detected to have folder-by-type, showing an example of how it should look, and linking to the [adr][tag-adr], which documents why this was chosen.

```text
It looks like 'services/catalogue/src' is using this folder structure, known as
"folder-by-type", which is discouraged:

services/catalogue/src
├── controllers
├── models
└── views

Instead, you should use folder-by-feature:

services/catalogue/src
├── details
│   ├── controller.ts
│   ├── model.ts
│   └── view.ts
├── indexing
└── search

For more information, see this ADR: ./docs/arch/005-folder-layout.md

If this test failure is a false positive, please let us know, or you can either
improve the test or add your folder path to the '.architecture-ignore' file.
Here is the fragment that can be added:

```toml
[[services]]

[service.catalogue]
allowFolderByType = true
```.

```

There is also the text on how to skip a test if there is a good reason to or the test failure is a false negative.  Adding to the `.architecture-ignore` file notifies the core team about an addition, but **does not block the PR**, as teams are all trusted; we just want to verify if something is happening a lot or if there is some case the tests are not handling.

An example of a good reason for ignoring this test is when a team is taking ownership of a service and adding it to the repository: they want to pull its source in and make as few changes as possible until it is under their control; refactoring can then happen later.

## Testing a Project Rule

Now let's look at how we verify that our services don't reference other services.  The test is similar to the previous one other than the `repositoryServices()` function returns a map of service names and Services.  The `Service` struct is an abstraction which allows us to handle both NodeJS projects and Go projects with the same test.

```go
func TestServicesCannotReferenceOtherServices(t *testing.T) {
  allServices := repositoryServices()

  for _, service := range allServices {

    for _, reference := range service.References {
      if other, found := allServices[reference.Name]; found {

         assert.Failf(t, "service references another service", wrap80(`
It looks like the '%s' service is referencing the '%s' service.

1.  Service Boundary
Needing data from another service is often an indication of non-optimal service boundary, which could mean we need to refactor our design a bit.

1.  Distributed Ball of Mud
Having many service to service dependencies make all our services more tightly coupled, making refactoring and deployment harder.

Sometimes a service to service reference is fine however!  You can add your service to service definition to the '.architecture-ignore' file if this is the case.  Here is the fragment that can be added:

%s
         `, service.Name, other.Name, serviceToServiceArchitectureIgnore(service, other)))
      }

    }
  }
```

The error message when rendered looks like this, again adding as much detail as we can along with how to add the exception if needed:

```md
It looks like the catalogue 'service' is referencing the 'offers' service.

Service to Service references are discouraged for two main reasons:

1.  Service Boundary
Needing data from another service is often an indication of non-optimal service
boundary, which could mean we need to refactor our design a bit.

2.  Distributed Ball of Mud
Having many service to service dependencies make all our services more tightly
coupled, making refactoring and deployment harder.

Sometimes a service to service reference is fine however!  You can add your
service to service definition to the '.architecture-ignore' file if this is the
case.  Here is the fragment that can be added:

```toml
[[services]]

[services.catalogue]
references = [
    "offers"
]
```.

```

## Further Work

Using tests like this also allows you to build extra things on top of them; for migrating from one library to another, you can add tests that specify that the number of usages can only go down over time, never up.

You can also use `codeowners` (or equivalent) to keep an eye on what is being added to the `.architecture-ignore` file, allowing you to react to emerging patterns and either guide teams towards the pattern or away from it.

The key with this is that you trust your teams; this is all "trust but verify" with the ignore file.  You should (almost) never be blocking a team from working.

[tag-adr]: /tags/adr/