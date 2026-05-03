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
  manualLabel: string;
  findIconLabel: string;
  unavailableHint: string;
  referenceHint: string;
  captureReady: string;
  captureHint: string;
  coordinateHint: string;
  pickMouse: string;
  onCoordinateSourceChange: (value: ClickCoordinateSource) => void;
  onStartCapture: () => void;
  onXChange: (value: string) => void;
  onYChange: (value: string) => void;
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
  const {
    coordinateSource,
    canUseFindIconReference,
    x,
    y,
    captureActive,
    captureLoading,
    manualLabel,
    findIconLabel,
    unavailableHint,
    referenceHint,
    captureReady,
    captureHint,
    coordinateHint,
    pickMouse,
    onCoordinateSourceChange,
    onStartCapture,
    onXChange,
    onYChange,
  } = props;
  const manual = coordinateSource === CLICK_COORDINATE_SOURCE_MANUAL;

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <label className="block text-xs font-medium text-gray-700">坐标来源</label>
        <select
          value={coordinateSource}
          onChange={(event) => onCoordinateSourceChange(event.target.value === CLICK_COORDINATE_SOURCE_FIND_ICON ? CLICK_COORDINATE_SOURCE_FIND_ICON : CLICK_COORDINATE_SOURCE_MANUAL)}
          className="h-10 w-full cursor-pointer appearance-none border border-gray-200 bg-white px-3 text-sm outline-none transition-colors hover:border-gray-300 focus:border-black focus:ring-1 focus:ring-black"
        >
          <option value={CLICK_COORDINATE_SOURCE_MANUAL}>{manualLabel}</option>
          {canUseFindIconReference ? <option value={CLICK_COORDINATE_SOURCE_FIND_ICON}>{findIconLabel}</option> : null}
        </select>
        <p className="text-[11px] text-gray-500">{canUseFindIconReference ? referenceHint : unavailableHint}</p>
      </div>
      {manual ? (
        <>
          <div className="flex items-center justify-between gap-3">
            <label className="block text-xs font-medium text-gray-700">坐标参数</label>
            <button
              type="button"
              onClick={onStartCapture}
              disabled={captureActive || captureLoading}
              className="border border-gray-300 bg-white px-3 py-1.5 text-[11px] font-bold text-gray-700 transition-colors hover:border-black hover:text-black disabled:cursor-not-allowed disabled:opacity-45"
            >
              {captureActive ? captureReady : pickMouse}
            </button>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <CoordinateInput axis="X" value={x} onChange={onXChange} />
            <CoordinateInput axis="Y" value={y} onChange={onYChange} />
          </div>
          <p className="text-[11px] text-gray-500">{captureActive ? captureHint : coordinateHint}</p>
        </>
      ) : null}
    </div>
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
