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

  nameDefMatches = funcString.match(/function .*\(/g)
  if (nameDefMatches === null) {
    console.log(JSON.stringify({'type': 'function_loaded', 'data': false}))
    return
  }

  defPos = funcString.indexOf(nameDefMatches)
  postDef = funcString.substring(defPos + nameDefMatches[0].length, funcString.length)

  argsString = postDef.substring(0, postDef.indexOf(')'))
  argsNoSpaces = argsString.replace(/ /g, "")
  args = argsNoSpaces.split(',')

  postArgs = postDef.substring(postDef.indexOf(')') + 1, postDef.length)
  body = postArgs.substring(postArgs.indexOf('{') + 1, postArgs.lastIndexOf('}'))

  if (args === null || body === null) {
    console.log(JSON.stringify({'type': 'function_loaded', 'data': false}))
    return
  }

  var f = new Function(args, body)
  if (f === null) {
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

    result = f(request['data'])
    console.log(JSON.stringify({'type': 'response', 'data': result}))
  })
})
