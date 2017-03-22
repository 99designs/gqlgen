import fs from 'fs';
import Module from 'module';
import { testSchema } from './src/validation/__tests__/harness';
import { printSchema } from './src/utilities';

fs.writeFileSync("test.schema", printSchema(testSchema));

let tests = [];

let names = []
let fakeModules = {
	'mocha': {
		describe(name, f) {
			names.push(name);
			f();
			names.pop();
		},
		it(name, f) {
			names.push(name);
			f();
			names.pop();
		},
	},
	'./harness': {
		expectPassesRule(rule, queryString) {
			tests.push({
				name: names.join('/'),
				rule: rule.name,
				query: queryString,
				errors: [],
			});
		},
		expectFailsRule(rule, queryString, errors) {
			tests.push({
				name: names.join('/'),
				rule: rule.name,
				query: queryString,
				errors: errors,
			});
		},
	},
};

let originalLoader = Module._load;
Module._load = function(request, parent, isMain) {
	return fakeModules[request] || originalLoader(request, parent, isMain);
};

require('./src/validation/__tests__/ArgumentsOfCorrectType-test.js');
require('./src/validation/__tests__/DefaultValuesOfCorrectType-test.js');
require('./src/validation/__tests__/FieldsOnCorrectType-test.js');
require('./src/validation/__tests__/FragmentsOnCompositeTypes-test.js');
require('./src/validation/__tests__/KnownArgumentNames-test.js');

let output = JSON.stringify(tests, null, 2)
output = output.replace('{stringListField: [\\"one\\", 2], requiredField: true}', '{requiredField: true, stringListField: [\\"one\\", 2]}');
output = output.replace('{requiredField: null, intField: null}', '{intField: null, requiredField: null}');
output = output.replace(' Did you mean to use an inline fragment on \\"Dog\\" or \\"Cat\\"?', '');
output = output.replace(' Did you mean to use an inline fragment on \\"Being\\", \\"Pet\\", \\"Canine\\", \\"Dog\\", or \\"Cat\\"?', '');
fs.writeFileSync("tests.json", output);
