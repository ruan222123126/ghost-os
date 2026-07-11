import {
  coerceStartInputDefaultText,
  formatStartInputJSON,
  isNumberInputText,
  parseStartInputDefaultErrorCode,
  parseStartInputDefault,
  validateStartInputDefaultCode,
  validateStartInputDefault,
} from '@/lib/workflow-editor/startInputDefaults';

describe('startInputDefaults', () => {
  it('parses boolean defaults from true/false text', () => {
    expect(parseStartInputDefault('boolean', 'true')).toBe(true);
    expect(parseStartInputDefault('boolean', 'false')).toBe(false);
  });

  it('rejects invalid number defaults', () => {
    expect(() => parseStartInputDefault('number', 'abc')).toThrow('value must be a finite number');
    expect(() => parseStartInputDefault('number', '')).toThrow('value must be a finite number');
  });

  it('validates object/array defaults with clear messages', () => {
    expect(validateStartInputDefault('object', '{"key":"value"}')).toBe('');
    expect(validateStartInputDefault('array', '{"key":"value"}')).toBe('value for array must be a JSON array');
  });

  it('returns stable error codes for invalid defaults', () => {
    expect(validateStartInputDefaultCode('number', 'abc')).toBe('START_INPUT_DEFAULT_NUMBER');
    expect(validateStartInputDefaultCode('object', '[1,2]')).toBe('START_INPUT_DEFAULT_OBJECT');
    expect(validateStartInputDefaultCode('array', '{}')).toBe('START_INPUT_DEFAULT_ARRAY');
    expect(validateStartInputDefaultCode('object', '{"key"')).toBe('START_INPUT_DEFAULT_JSON');
  });

  it('extracts error code from parse failures', () => {
    try {
      parseStartInputDefault('object', '{"key"');
      throw new Error('expected parse failure');
    } catch (error) {
      expect(parseStartInputDefaultErrorCode(error)).toBe('START_INPUT_DEFAULT_JSON');
    }
  });

  it('formats object JSON defaults on blur', () => {
    const formatted = formatStartInputJSON('object', '{"key":"value","count":1}');
    expect(formatted).toBe('{\n  "key": "value",\n  "count": 1\n}');
  });

  it('coerces boolean defaults to true/false options', () => {
    expect(coerceStartInputDefaultText('boolean', 'true')).toBe('true');
    expect(coerceStartInputDefaultText('boolean', '')).toBe('false');
  });

  it('accepts only numeric draft text for number input', () => {
    expect(isNumberInputText('12')).toBe(true);
    expect(isNumberInputText('-12.5')).toBe(true);
    expect(isNumberInputText('.5')).toBe(true);
    expect(isNumberInputText('1e3')).toBe(false);
    expect(isNumberInputText('abc')).toBe(false);
  });
});
