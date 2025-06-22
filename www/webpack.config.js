const CopyWebpackPlugin = require("copy-webpack-plugin");
const path = require('path');

module.exports = {
  entry: "./bootstrap.js",
  output: {
    path: path.resolve(__dirname, "dist"),
    filename: "bootstrap.js",
  },
  mode: "development",
  plugins: [

    new CopyWebpackPlugin({
      patterns: [
        { from: 'index.html', to: 'index.html' },
        { from: 'auth.html', to: 'auth.html' },
        { from: 'checkmate.css', to: 'checkmate.css' },
        { from:'src/auth.css', to: 'auth.css' },
        { from: 'pieces/', to: 'pieces' },
        { from: 'src/auth.js', to: 'auth.js' },
        { from: 'src/main.js', to: 'main.js' },
        { from:'lobby.html', to:'lobby.html' },
        { from:'src/lobby.css', to:'lobby.css' },
        { from:'src/lobby.js', to:'lobby.js' },
      ]
    })
  ],
  experiments: {
    asyncWebAssembly: true,
  },
  module: {
    rules: [
      {
        test: /\.wasm$/,
        type: "webassembly/async",
      },
    ],
  },
};