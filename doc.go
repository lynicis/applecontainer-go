/*
Package applecontainer implements a testcontainers-go-style Go library for spinning up
Apple Container CLI Linux containers as test dependencies.

Example Usage:

	ctx := context.Background()
	container, err := applecontainer.Run(ctx, "nginx:alpine",
		applecontainer.WithExposedPorts("80"),
		applecontainer.WithWaitStrategy(wait.ForHTTP("/")),
	)
	if err != nil {
		log.Fatalf("failed to run: %v", err)
	}
	defer container.Terminate(ctx)

	endpoint, err := container.Endpoint(ctx, "80")
	// HTTP request against endpoint...
*/
package applecontainer
