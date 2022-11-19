import { Greeter, HelloRequest } from "./services/greeter.pb";

const name = prompt("What's your name?") || "stranger";
const resp = await Greeter.SayHello({ name });
alert(resp.message);
