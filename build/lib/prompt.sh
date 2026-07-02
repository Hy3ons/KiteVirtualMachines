#!/usr/bin/env bash

kite_prompt_interactive() {
  [[ -t 0 && "${KITE_ASSUME_DEFAULTS:-false}" != "true" ]]
}

kite_prompt_bool() {
  local prompt="$1"
  local default_value="$2"
  local default_label
  local answer

  if [[ "${default_value}" == "true" ]]; then
    default_label="1"
  else
    default_label="2"
  fi

  while true; do
    printf '%s\n' "${prompt}" >&2
    printf '  1) 예\n' >&2
    printf '  2) 아니오\n' >&2
    read -r -p "선택 [1/2, 기본: ${default_label}] " answer
    answer="${answer:-${default_label}}"

    case "${answer}" in
      1|y|Y|yes|YES)
        return 0
        ;;
      2|n|N|no|NO)
        return 1
        ;;
      *)
        printf '[kite] WARNING: 1 또는 2를 입력하세요\n' >&2
        ;;
    esac
  done
}

kite_prompt_configure_bool() {
  local variable_name="$1"
  local was_set="$2"
  local prompt="$3"
  local current_value

  eval "current_value=\"\${${variable_name}:-false}\""

  if [[ -n "${was_set}" || ! kite_prompt_interactive ]]; then
    return 0
  fi

  if kite_prompt_bool "${prompt}" "${current_value}"; then
    printf -v "${variable_name}" '%s' "true"
  else
    printf -v "${variable_name}" '%s' "false"
  fi
  export "${variable_name}"
}
