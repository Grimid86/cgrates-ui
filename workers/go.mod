module github.com/Grimid86/cgrates-ui/workers

go 1.22

require (
	github.com/Grimid86/cgrates-ui/backend v0.0.0
	github.com/apache/pulsar-client-go v0.12.1
)

replace github.com/Grimid86/cgrates-ui/backend => ../backend
