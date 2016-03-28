The Jenkins build log tailer
============================

`tailbuild` is a small program that will tail in real-time the logs
of any project in your Jenkins build server.

The way you use it is straightforward — after starting your build:

```
tailbuild <project>
````

As the project builds, `tailbuild` will research all log files
of your Jenkins build, and show them to you in real-time.
