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
			switch (name) {
			case 'within schema language':
				return;
			}
			names.push(name);
			f();
			names.pop();
		},
		it(name, f) {
			switch (name) {
			case 'ignores type definitions':
				return;
			}
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

require('./src/validation/__tests__/ArgumentsOfCorrectType-test');
require('./src/validation/__tests__/DefaultValuesOfCorrectType-test');
require('./src/validation/__tests__/FieldsOnCorrectType-test');
require('./src/validation/__tests__/FragmentsOnCompositeTypes-test');
require('./src/validation/__tests__/KnownArgumentNames-test');
require('./src/validation/__tests__/KnownDirectives-test');
require('./src/validation/__tests__/KnownFragmentNames-test');
require('./src/validation/__tests__/KnownTypeNames-test');
require('./src/validation/__tests__/LoneAnonymousOperation-test');
require('./src/validation/__tests__/NoFragmentCycles-test');
// require('./src/validation/__tests__/NoUndefinedVariables-test');
require('./src/validation/__tests__/NoUnusedFragments-test');
// require('./src/validation/__tests__/NoUnusedVariables-test');
// require('./src/validation/__tests__/OverlappingFieldsCanBeMerged-test');
require('./src/validation/__tests__/PossibleFragmentSpreads-test');
require('./src/validation/__tests__/ProvidedNonNullArguments-test');
require('./src/validation/__tests__/ScalarLeafs-test');
require('./src/validation/__tests__/UniqueArgumentNames-test');
require('./src/validation/__tests__/UniqueDirectivesPerLocation-test');
require('./src/validation/__tests__/UniqueFragmentNames-test');
require('./src/validation/__tests__/UniqueInputFieldNames-test');
require('./src/validation/__tests__/UniqueOperationNames-test');
require('./src/validation/__tests__/UniqueVariableNames-test');
require('./src/validation/__tests__/VariablesAreInputTypes-test');
// require('./src/validation/__tests__/VariablesInAllowedPosition-test');

let output = JSON.stringify(tests, null, 2)
output = output.replace(' Did you mean to use an inline fragment on \\"Dog\\" or \\"Cat\\"?', '');
output = output.replace(' Did you mean to use an inline fragment on \\"Being\\", \\"Pet\\", \\"Canine\\", \\"Dog\\", or \\"Cat\\"?', '');
output = output.replace(' Did you mean \\"Pet\\"?', '');
fs.writeFileSync("tests.json", output);
