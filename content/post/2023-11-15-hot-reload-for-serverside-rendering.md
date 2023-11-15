+++
title = 'Hot Reload for ServerSide Rendering'
tags = ['developer experience', 'productivity']
+++


In one of my too many side projects, I am using [htmx] and go templates to render a somewhat complicated web UI.  I much prefer using htmx for this kind of thing rather than react, as react brings in so much more additional complexity than I need or want.  However, there is one thing I miss from the React ecosystem, and that is hot reload.

Being able to save a file in my editor and see the changes instantly in a web browser is an amazing developer experience, and I want to recreate that for htmx.  I realised the steps to build my own hot reload were actually pretty small.

On the server side:
* generate a guid on startup
* expose this to the client somehow

On the client side:
* fetch the guid
* if the guid doesn't match what we have seen before, refresh the page

Despite my preference for HTMX and html/template in Go, neither the Client nor Server implementations a framework specific.  The server utilises [Fiber] as its host, but it is not a hard requirement.

## The Client

I decided to use a websocket for the transport, as if I decide later to make the server notify the client of changes also.  For the client side, I have a single script that I include in the html template, which connects a websocket, and handles all messages received.  It also handles reconnection if the server disconnects, along with a simple backoff mechanism.

```js
(function () {
  var lastUuid = "";
  var timeout;

  const resetBackoff = () => {
    timeout = 1000;
  };

  const backOff = () => {
    if (timeout > 10 * 1000) {
      return;
    }

    timeout = timeout * 2;
  };

  const hotReloadUrl = () => {
    const hostAndPort =
      location.hostname + (location.port ? ":" + location.port : "");

    if (location.protocol === "https:") {
      return "wss://" + hostAndPort + "/ws/hotreload";
    } else if (location.protocol === "http:") {
      return "ws://" + hostAndPort + "/ws/hotreload";
    }
  };

  function connectHotReload() {
    const socket = new WebSocket(hotReloadUrl());

    socket.onmessage = (event) => {
      if (lastUuid === "") {
        lastUuid = event.data;
      }

      if (lastUuid !== event.data) {
        console.log("[Hot Reloader] Server Changed, reloading");
        location.reload();
      }
    };

    socket.onopen = () => {
      resetBackoff();
      socket.send("Hello");
    };

    socket.onclose = () => {
      const timeoutId = setTimeout(function () {
        clearTimeout(timeoutId);
        backOff();

        connectHotReload();
      }, timeout);
    };
  }

  resetBackoff();
  connectHotReload();
})();
```

Note this is a pretty dumb hot reload - it just refreshes the current page.

## The Server

The entire implementation is about 20 lines of go, utilising the `websocket` package for `fiber`, my web server framework of choice.  There is not a lot to it; just create a UUID, and send that value to any client which connects to the websocket, and sends any message to us.


```go
import (
  "github.com/gofiber/contrib/websocket"
  "github.com/gofiber/fiber/v2"
  "github.com/google/uuid"
)

func WithHotReload(app *fiber.App) {
  id := []byte(uuid.New().String())

  app.Use("/ws", func(c *fiber.Ctx) error {
    if websocket.IsWebSocketUpgrade(c) {
      return c.Next()
    }
    return fiber.ErrUpgradeRequired
  })

  app.Get("/ws/hotreload", websocket.New(func(c *websocket.Conn) {
    for {
      if _, _, err := c.ReadMessage(); err != nil {
        break
      }

      if err := c.WriteMessage(websocket.TextMessage, id); err != nil {
        break
      }
    }
  }))

}
```


## How it works

I use the [modd] tool to restart my go applications when I am developing them: any time I save a file, the app restarts.

When the app restarts, all websocket connections are aborted.  The client then tries to reconnect, and when it does, it receives a new UUID from the server, causing the entire page to refresh.  As all my apps are serverside rendered, there is usually little, if any, state to keep, so a full page refresh is fine for my development needs.

In this implementation, the socket is not needed; it would be just as easy to poll an API every x seconds to see if the UUID has changed, but having the client react to the server breaking the connection on restart seems better; there is less random HTTP noise in the network tab too.

## Future modifications

I think I could make htmx do the hard work of switching out the page dom when the socket indicates the page has changed, and while that would be cool, it does mean that the client part would become htmx specific, so I probably won't do this.

[modd]: https://github.com/cortesi/modd/
[htmx]: https://htmx.org/
[fiber]: https://gofiber.io/