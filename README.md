# errmux
Concurrent error handling utilities for Go.

# Description
errmux provides functions and data for working with concurrent error handling
in a similar fashion to serial error handling. Unlike most error handling packages,
errmux operates on channels of errors rather than errors themselves.

When a new Handler is created, errmux multiplexes all of its constituent error
channel and passes the values to a Consumer for processing. All values on all error
channels will be read - this is to prevent resource leaks due to blocking send
operations upstream from a Handler. Errors will only be passed to a Consumer
after reading if the following conditions apply:

1. Consumer has not previously reported that it is closed (`c.Consume(err) == false`)
2. `Cancel` has not been called on a Handler

If a Handler has been canceled, this does not guarantee further errors will not be
read by the Handler. It does, however, guarantee errors will not be consumed by a
Consumer.

