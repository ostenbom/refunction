var readline = require("readline")
var rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

console.log(JSON.stringify({"type": "started", "data": ""}))

var loaded = false
var f = null

rl.on('line', function(line){
  try {
    request = JSON.parse(line)
  } catch(error) {
    console.error(error)
    return
  }

  if (request['type'] !== 'function') {
    return
  }

  funcString = request['data']

  var Module = module.constructor
  var m = new Module()
  m._compile(funcString, '/tmp/none')
  user_exports = m.exports;

  if (user_exports === null) {
    console.log(JSON.stringify({'type': 'function_loaded', 'data': false}))
    return
  }

  console.log(JSON.stringify({'type': 'function_loaded', 'data': true}))

  rl.close()
  var rl2 = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    terminal: false
  });

  rl2.on('line', function(line) {
    try {
      request = JSON.parse(line)
    } catch(error) {
      console.error(error)
      return
    }

    if (request['type'] !== 'request') {
      return;
    }

    result = user_exports.handler(request['data'])
    console.log(JSON.stringify({'type': 'response', 'data': result}))
  })
})
