const path = require('path')
const HtmlWebpackPlugin = require('html-webpack-plugin')
const CopyPlugin = require('copy-webpack-plugin')
const webpack = require('webpack')
const packageJson = require('./package.json')

const { ModuleFederationPlugin } = webpack.container

const COMPONENT_NAME = 'intro_course_developer_component'
const COMPONENT_DEV_PORT = 3005

module.exports = (env = {}) => {
  const getVariable = (name) => env[name]
  const IS_DEV = getVariable('NODE_ENV') !== 'production'
  const deps = packageJson.dependencies || {}

  return {
    target: 'web',
    mode: IS_DEV ? 'development' : 'production',
    devtool: IS_DEV ? 'source-map' : undefined,
    entry: './src/index.js',
    devServer: {
      static: {
        directory: path.join(__dirname, 'public'),
      },
      compress: true,
      hot: true,
      historyApiFallback: true,
      port: COMPONENT_DEV_PORT,
      client: {
        progress: true,
      },
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
      filename: '[name].[contenthash].js',
      path: path.resolve(__dirname, 'build'),
      publicPath: 'auto',
    },
    resolve: {
      extensions: ['.ts', '.tsx', '.js', '.mjs', '.jsx'],
      alias: {
        '@': path.resolve(__dirname, '../shared_library'),
      },
    },
    plugins: [
      new ModuleFederationPlugin({
        name: COMPONENT_NAME,
        filename: 'remoteEntry.js',
        exposes: {
          './routes': './routes',
          './sidebar': './sidebar',
          './provide': './src/provide',
        },
        shared: {
          react: { singleton: true, requiredVersion: deps.react || false },
          'react-dom': { singleton: true, requiredVersion: deps['react-dom'] || false },
          'react-router-dom': { singleton: true, requiredVersion: deps['react-router-dom'] || false },
          '@tanstack/react-query': {
            singleton: true,
            requiredVersion: deps['@tanstack/react-query'] || false,
          },
          '@tumaet/prompt-shared-state': {
            singleton: true,
            requiredVersion: deps['@tumaet/prompt-shared-state'] || false,
          },
        },
      }),
      new CopyPlugin({
        patterns: [{ from: 'public' }],
      }),
      new HtmlWebpackPlugin({
        template: 'public/template.html',
        minify: {
          removeComments: true,
          collapseWhitespace: true,
          removeRedundantAttributes: true,
          useShortDoctype: true,
          removeEmptyAttributes: true,
          removeStyleLinkTypeAttributes: true,
          keepClosingSlash: true,
          minifyJS: true,
          minifyCSS: true,
          minifyURLs: true,
        },
      }),
    ],
    cache: {
      type: 'filesystem',
    },
  }
}
