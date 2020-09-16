require 'sinatra'
require 'yaml'

configure do
  set :bind, "0.0.0.0"
end

get '/' do
  'Hello world!'
end

get '/yaml' do
  '<pre>' + YAML.load("{foo: [bar, baz, quux]}").to_yaml + '</pre>'
end

get '/env' do
  rval = "<pre>"
  ENV.keys.sort.each do |key|
    rval += "#{key}: #{ENV[key]}\n"
  end
  rval += "</pre>"
  rval
end
