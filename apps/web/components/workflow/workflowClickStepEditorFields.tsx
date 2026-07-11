'use client';

import {
  CLICK_COORDINATE_SOURCE_FIND_ICON,
  CLICK_COORDINATE_SOURCE_MANUAL,
  POSITION_TYPE_ABSOLUTE,
  POSITION_TYPE_RELATIVE,
  type ClickCoordinateSource,
  type ClickPositionType,
} from '@/components/workflow/workflowClickStepEditorHelpers';

interface PositionTypeFieldProps {
  value: ClickPositionType;
  onChange: (value: ClickPositionType) => void;
}

interface CoordinateEditorFieldsProps {
  coordinateSource: ClickCoordinateSource;
  canUseFindIconReference: boolean;
  x: string;
  y: string;
  captureActive: boolean;
  captureLoading: boolean;
  text: CoordinateEditorText;
  onCoordinateSourceChange: (value: ClickCoordinateSource) => void;
  onStartCapture: () => void;
  onXChange: (value: string) => void;
  onYChange: (value: string) => void;
}

interface CoordinateEditorText {
  captureHint: string;
  captureReady: string;
  coordinateHint: string;
  findIconReference: string;
  findIconReferenceHint: string;
  findIconUnavailable: string;
  manualCoordinates: string;
  pickMouse: string;
}

export function ClickEditorPositionTypeField(props: PositionTypeFieldProps) {
  const { value, onChange } = props;
  return (
    <div className="space-y-2">
      <label className="block text-xs font-medium text-gray-700">位置基准</label>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value === POSITION_TYPE_ABSOLUTE ? POSITION_TYPE_ABSOLUTE : POSITION_TYPE_RELATIVE)}
        className="h-10 w-full cursor-pointer appearance-none border border-gray-200 bg-white px-3 text-sm outline-none transition-colors hover:border-gray-300 focus:border-black focus:ring-1 focus:ring-black"
      >
        <option value={POSITION_TYPE_RELATIVE}>相对于鼠标原本位置 (Relative)</option>
        <option value={POSITION_TYPE_ABSOLUTE}>屏幕绝对位置 (Absolute)</option>
      </select>
      <p className="text-[11px] text-gray-500">
        <span className="text-gray-800">必填</span> · enum · 决定坐标系原点.
      </p>
    </div>
  );
}

export function ClickEditorCoordinateFields(props: CoordinateEditorFieldsProps) {
  return (
    <div className="space-y-4">
      <CoordinateSourceField
        canUseFindIconReference={props.canUseFindIconReference}
        coordinateSource={props.coordinateSource}
        text={props.text}
        onChange={props.onCoordinateSourceChange}
      />
      <ManualCoordinateFields
        active={props.coordinateSource === CLICK_COORDINATE_SOURCE_MANUAL}
        captureActive={props.captureActive}
        captureLoading={props.captureLoading}
        text={props.text}
        x={props.x}
        y={props.y}
        onStartCapture={props.onStartCapture}
        onXChange={props.onXChange}
        onYChange={props.onYChange}
      />
    </div>
  );
}

function CoordinateSourceField(props: {
  canUseFindIconReference: boolean;
  coordinateSource: ClickCoordinateSource;
  text: CoordinateEditorText;
  onChange: (value: ClickCoordinateSource) => void;
}) {
  const { canUseFindIconReference, coordinateSource, text, onChange } = props;
  return (
    <div className="space-y-2">
      <label className="block text-xs font-medium text-gray-700">坐标来源</label>
      <select
        value={coordinateSource}
        onChange={(event) => onChange(parseCoordinateSourceOption(event.target.value))}
        className="h-10 w-full cursor-pointer appearance-none border border-gray-200 bg-white px-3 text-sm outline-none transition-colors hover:border-gray-300 focus:border-black focus:ring-1 focus:ring-black"
      >
        <option value={CLICK_COORDINATE_SOURCE_MANUAL}>{text.manualCoordinates}</option>
        {canUseFindIconReference ? <option value={CLICK_COORDINATE_SOURCE_FIND_ICON}>{text.findIconReference}</option> : null}
      </select>
      <p className="text-[11px] text-gray-500">
        {canUseFindIconReference ? text.findIconReferenceHint : text.findIconUnavailable}
      </p>
    </div>
  );
}

function ManualCoordinateFields(props: {
  active: boolean;
  captureActive: boolean;
  captureLoading: boolean;
  text: CoordinateEditorText;
  x: string;
  y: string;
  onStartCapture: () => void;
  onXChange: (value: string) => void;
  onYChange: (value: string) => void;
}) {
  const { active, captureActive, captureLoading, text, x, y, onStartCapture, onXChange, onYChange } = props;
  if (!active) {
    return null;
  }
  return (
    <>
      <div className="flex items-center justify-between gap-3">
        <label className="block text-xs font-medium text-gray-700">坐标参数</label>
        <CaptureMouseButton
          captureActive={captureActive}
          captureLoading={captureLoading}
          text={text}
          onStartCapture={onStartCapture}
        />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <CoordinateInput axis="X" value={x} onChange={onXChange} />
        <CoordinateInput axis="Y" value={y} onChange={onYChange} />
      </div>
      <p className="text-[11px] text-gray-500">{captureActive ? text.captureHint : text.coordinateHint}</p>
    </>
  );
}

function CaptureMouseButton(props: {
  captureActive: boolean;
  captureLoading: boolean;
  text: CoordinateEditorText;
  onStartCapture: () => void;
}) {
  const { captureActive, captureLoading, text, onStartCapture } = props;
  return (
    <button
      type="button"
      onClick={onStartCapture}
      disabled={captureActive || captureLoading}
      className="border border-gray-300 bg-white px-3 py-1.5 text-[11px] font-bold text-gray-700 transition-colors hover:border-black hover:text-black disabled:cursor-not-allowed disabled:opacity-45"
    >
      {captureActive ? text.captureReady : text.pickMouse}
    </button>
  );
}

function CoordinateInput(props: { axis: 'X' | 'Y'; value: string; onChange: (value: string) => void }) {
  const { axis, value, onChange } = props;
  return (
    <div className="flex bg-gray-50/50">
      <span className="flex w-12 flex-none items-center justify-center border border-r-0 border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-600">
        {axis}
      </span>
      <input
        type="number"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="0"
        className="w-full border border-gray-200 bg-white px-3 py-2 text-sm outline-none transition-colors focus:border-black focus:ring-1 focus:ring-black"
      />
    </div>
  );
}

function parseCoordinateSourceOption(value: string): ClickCoordinateSource {
  if (value === CLICK_COORDINATE_SOURCE_FIND_ICON) {
    return CLICK_COORDINATE_SOURCE_FIND_ICON;
  }
  return CLICK_COORDINATE_SOURCE_MANUAL;
}
