#!/usr/bin/nodejs

var http = require('https');
var proc = require('child_process');
var async = require("async");
var HOST = null;
var TOKEN = null;

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

function waitForNanocloudToBeOnline() {
  var command = 'curl --output /dev/null --insecure --silent --write-out \'%{http_code}\n\' "https://$(docker exec proxy hostname -I | awk \'{print $1}\')"';

  console.log("Try to connect")
  try {
    var returnedValue = proc.execSync(command);
  } catch (e) {
    waitForNanocloudToBeOnline();
  }

  if (returnedValue.toString() == "200\n") {
    console.log("Nanocloud available");

    return ;
  }

  waitForNanocloudToBeOnline();
}

function setHost() {
  var command = "docker exec proxy hostname -I | awk \'{print $1}\'";

  console.log("Determining host")
  var returnedValue = proc.execSync(command);

  HOST = returnedValue.toString().trim();
  console.log("Host address: " + HOST)
}

function request(options, callback) {
  var message = "";
  var status = null;

  var headers = options.headers || {};
  var param = JSON.stringify(options.param) || "";

  if (TOKEN) {
    headers.Authorization = "Bearer " + TOKEN
  }

  var options = {
    host: HOST,
    path: options.path,
    method: options.verb,
    port: 443,
    headers: headers
  };

  var req = http.request(options, function(res) {
    status = res.statusCode;

    res.setEncoding('utf8');
    res.on('data', function(chunk) {
      message += chunk;
    });
    res.on('end', function() {
      console.log("Call to " + options.path + " returned " + status)
      if (callback && status == 200)
        callback(JSON.parse(message));
      if (status != 200)
        console.log(message);
    })
  });

  req.on('error', function(e) {
    console.log('problem with request: ' + e.message);
  });

  req.write(param);
  req.end();
}

function bootWindows(next) {
  request({
    path: '/api/iaas/windows-custom-server-127.0.0.1-windows-server-std-2012R2-amd64/start',
    verb: 'POST'
  }, function() {
    if (next)
      next()
  });
}

function waitForWindowsToBeRunning(next) {
  console.log("Wainting for Windows to be running....");
  request({
    path: '/api/iaas',
    verb: 'GET',
  }, function(res) {
    if (res.data[0].attributes.status != "running") {
      waitForWindowsToBeRunning();
    }
    else
      if (next)
        next();
  });
}

function login(next) {
  request({
    path: '/oauth/token',
    verb: 'POST',
    param: {
      username: "admin@nanocloud.com",
      password: "admin",
      grant_type: "password"
    },
    headers: {
      'Content-Type': 'application/json',
      'Authorization': "Basic OTQwNWZiNmIwZTU5ZDI5OTdlM2M3NzdhMjJkOGYwZTYxN2E5ZjViMzZiNjU2NWM3NTc5ZTViZTZkZWI4ZjdhZTo5MDUwZDY3YzJiZTA5NDNmMmM2MzUwNzA1MmRkZWRiM2FlMzRhMzBlMzliYmJiZGFiMjQxYzkzZjhiNWNmMzQx"
    }
  }, function(res) {
    TOKEN = res.access_token;
    console.log("Got token : " + TOKEN);
    if (next)
      next();
  });
}

function setHostInEnv() {
  var command = 'sed -i "s/value\\": \\"127.0.0.1\\"/value\\": \\"$(docker exec proxy hostname -I | awk \'{print $1}\')\\"/g" api/NanoEnv.postman_environment'

  console.log("Setting host in api file")
  var returnedValue = proc.execSync(command);

  console.log(returnedValue.toString());
}

function done() {
  console.log('Ready to perform tests')
}

waitForNanocloudToBeOnline();
setHost();

async.waterfall([
  setHostInEnv,
  login,
  bootWindows,
  waitForWindowsToBeRunning,
  done
])

