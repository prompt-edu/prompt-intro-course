/**
 * Minimal webpack config for screenshot harness.
 * Skips Module Federation (no @/ shared_library deps).
 */
const path = require('path')
const HtmlWebpackPlugin = require('html-webpack-plugin')

module.exports = {
  target: 'web',
  mode: 'development',
  devtool: 'source-map',
  entry: './src/screenshotEntry.tsx',
  devServer: {
    static: { directory: path.join(__dirname, 'public') },
    compress: true,
    hot: true,
    historyApiFallback: true,
    port: 3006,
    open: false,
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        use: {
          loader: 'ts-loader',
          options: {
            configFile: path.resolve(__dirname, 'tsconfig.json'),
            transpileOnly: true,
          },
        },
        exclude: /node_modules/,
      },
      {
        test: /\.css$/i,
        use: ['style-loader', 'css-loader', 'postcss-loader'],
        exclude: /node_modules/,
      },
      {
        test: /\.css$/i,
        include: /node_modules/,
        use: ['style-loader', 'css-loader'],
      },
    ],
  },
  output: {
    filename: 'screenshot.js',
    path: path.resolve(__dirname, 'build-screenshot'),
    publicPath: '/',
  },
  resolve: {
    extensions: ['.ts', '.tsx', '.js', '.mjs', '.jsx'],
    alias: {
      '@/env': path.resolve(__dirname, 'src/screenshotStubs/env.ts'),
      '@/utils/parseURL': path.resolve(__dirname, 'src/screenshotStubs/parseURL.ts'),
    },
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: 'public/template.html',
    }),
  ],
  cache: { type: 'filesystem' },
}
