![GitHub Release](https://img.shields.io/github/v/release/temporalio/temporal)
![Code Coverage](https://img.shields.io/badge/codecov-report-blue)
[![Community](https://img.shields.io/static/v1?label=community&message=get%20help&color=informational)](https://community.temporal.io)
[![Go Report Card][go-report-image]][go-report-url]

[go-report-image]: https://goreportcard.com/badge/github.com/temporalio/temporal
[go-report-url]: https://goreportcard.com/report/github.com/temporalio/temporal

# Temporal

Temporal is a durable execution platform that enables developers to build scalable applications without sacrificing productivity or reliability.
The Temporal server executes units of application logic called Workflows in a resilient manner that automatically handles intermittent failures, and retries failed operations.

Temporal is a mature technology that originated as a fork of Uber's Cadence.
It is developed by [Temporal Technologies](https://temporal.io/), a startup by the creators of Cadence.

[![image](https://github.com/temporalio/temporal/assets/251288/693d18b5-01de-4a3b-b47b-96347b84f610)](https://youtu.be/wIpz4ioK0gI 'Getting to know Temporal')

Learn more:

- [Courses](https://learn.temporal.io/courses/temporal_101/)
- [Docs](https://docs.temporal.io)
- Internal architecture: [docs/](./docs/architecture/README.md)

## Getting Started

### Download and Start Temporal Server Locally

Execute the following commands to start a pre-built image along with all the dependencies.

```bash
brew install temporal
temporal server start-dev
```

Refer to [Temporal CLI](https://docs.temporal.io/cli/#installation) documentation for more installation options.

### Run the Samples

Clone or download samples for [Go](https://github.com/temporalio/samples-go) or [Java](https://github.com/temporalio/samples-java) and run them with the local Temporal server.
We have a number of [HelloWorld type scenarios](https://github.com/temporalio/samples-java#helloworld) available, as well as more advanced ones. Note that the sets of samples are currently different between Go and Java.

### Use CLI

Use [Temporal CLI](https://docs.temporal.io/cli/) to interact with the running Temporal server.

```bash
temporal operator namespace list
temporal workflow list
```

### Use Temporal Web UI

Try [Temporal Web UI](https://docs.temporal.io/web-ui) by opening [http://localhost:8233](http://localhost:8233) for viewing your sample workflows executing on Temporal.

## Repository

This repository contains the source code of the Temporal server. To implement Workflows, Activities and Workers, use one of the [supported languages](https://docs.temporal.io/dev-guide/).

## Contributing

We'd love your help in making Temporal great.

Helpful links to get started:

- [work on or propose a new feature](https://github.com/temporalio/proposals)
- [learn about the Temporal Server architecture](./docs/architecture/README.md)
- [learn how to build and run the Temporal Server locally](./CONTRIBUTING.md)
- [learn about Temporal Server testing tools and best practices](./docs/development/testing.md)
- join the Temporal community [forum](https://community.temporal.io) and [Slack](https://t.mp/slack)

## License

[MIT License](https://github.com/temporalio/temporal/blob/main/LICENSE)
