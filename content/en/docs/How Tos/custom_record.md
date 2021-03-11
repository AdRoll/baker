---
title: "Create a custom Record"
date: 2020-10-29
weight: 300
description: >
  Record is the interface provided by Baker to represent an "object" of data
---

As you can read in the [Record and LogLine page](/docs/core-concepts/record_implementation/),
Baker processes objects called **records**.
A `Record`, in Baker, is an interface that provides an abstraction over a record of flattened data,
where columns of fields are indexed through integers.
If the Record implementations provided by Baker doesn't fit your needs, you can create your own
version, implementing the [`Record` inteface](https://pkg.go.dev/github.com/AdRoll/baker#Record).

## How to use a custom version of the Record

Once your Record version is ready, you need to use it in your code.
To do so, you must create and fill a [`baker.Components`](/docs/how-tos/baker_components/) struct.
The only required field to set is the `CreateRecord`, which should return a new instance of your Record 
(see more details at [CreateRecord](/docs/how-tos/baker_components/#createrecord)).

```go
comp := baker.Components{
	CreateRecord: func() baker.Record {
		return &MyCustomRecord{}
	},

	// ... other configuration
}
```

Optionally, it is possible to set other functionality through the `baker.Components`. 
For instance, you can provide a custom [Validation](/docs/how-tos/baker_components/#validate) function 
or a custom [FieldByName](/docs/how-tos/baker_components/#fieldbyname) function for your own Record
implementation.

## Record conformance test

[`test_helper.go`](https://github.com/AdRoll/baker/blob/23938bc743100373379403dd25618c25f0822231/test_helper.go#L11)
provides a test helper, `RecordConformanceTest`, one can and should use to verify their 
custom `Record` satisfies the invariants required for any `Record` implementation.

Just pass to `RecordConformanceTest` a factory function creating new instances of your `Record`.

{{% alert title="Warning" color="warning" %}}
The conformance test provides a way to verify that a record implementation respects the
invariant that Baker requires for a Record implementation and thus it should always
be executed against all custom implementations of the Record.
{{% /alert %}}
