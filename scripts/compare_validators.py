"""Compare proposer assignments from four exports over a shared epoch window."""

import argparse
import csv
import json
from pathlib import Path


MODELS = ("baseline", "srsw", "lsw", "desw")
COLORS = ("#64748b", "#2563eb", "#ea580c", "#16a34a")
# This value appears without a stake in the exported data, outside the validator roster.
UNKNOWN_PROPOSER = 9223372036854775807


def read_csv(path):
    with path.open(encoding="utf-8-sig", newline="") as stream:
        return list(csv.DictReader(stream))


def load_comparison(input_dir, requested_epochs=None, selected_runs=None, allow_incomplete=False):
    selected_runs = selected_runs or {}
    runs = {}
    for path in sorted(input_dir.rglob("run_status.json")):
        status = json.loads(path.read_text(encoding="utf-8"))
        name = status.get("model")
        if name not in MODELS:
            continue
        if name in selected_runs:
            continue
        if name in runs:
            raise ValueError(f"Multiple exports for {name}: {runs[name][0]} and {path.parent}. "
                             f"Select one with --run {name}=<export-directory>.")
        runs[name] = (path.parent, status)
    for name, directory in selected_runs.items():
        status = json.loads((directory / "run_status.json").read_text(encoding="utf-8"))
        if status.get("model") != name:
            raise ValueError(f"{directory}: run_status.json does not identify model {name}.")
        runs[name] = (directory, status)
    missing = set(MODELS) - runs.keys()
    if missing:
        raise ValueError(f"Missing exports: {', '.join(sorted(missing))}")

    presets = {int(status["slots_per_epoch"]) for _, status in runs.values()}
    if len(presets) != 1 or min(presets) <= 0:
        raise ValueError("All models must have the same positive slots_per_epoch.")
    slots_per_epoch = presets.pop()
    available = min(int(status["comparable_epochs"]) for _, status in runs.values())
    epochs = available if requested_epochs is None else requested_epochs
    if epochs <= 0 or (epochs > available and not allow_incomplete):
        raise ValueError(f"Epoch count must be between 1 and {available}.")

    counts = {}
    rosters = {}
    unknown_slots = {}
    missing_slots = {}
    for name in MODELS:
        directory, _ = runs[name]
        roster = {int(row["validator_index"]) for row in read_csv(directory / "proposers_aggregated.csv")}
        roster.discard(UNKNOWN_PROPOSER)
        if not roster:
            raise ValueError(f"{name}: empty validator roster.")
        model_counts = dict.fromkeys(roster, 0)
        seen = set()
        unknown_slots[name] = []
        raw_rows = read_csv(directory / "proposers_raw.csv")
        if not raw_rows or max(int(row["epoch"]) for row in raw_rows) < epochs - 1:
            raise ValueError(f"{name}: export has not reached epoch {epochs - 1}.")
        for row in raw_rows:
            epoch, slot = int(row["epoch"]), int(row["slot"])
            if not 0 <= epoch < epochs:
                continue
            if row["model"] != name:
                raise ValueError(f"{name}: CSV contains a different model.")
            if slot in seen or slot // slots_per_epoch != epoch:
                raise ValueError(f"{name}: duplicate slot or inconsistent epoch at slot {slot}.")
            seen.add(slot)
            validator = int(row["validator_index"])
            if validator == UNKNOWN_PROPOSER:
                unknown_slots[name].append(slot)
                continue
            if validator not in roster:
                raise ValueError(f"{name}: validator {validator} is missing from aggregated CSV.")
            model_counts[validator] += 1
        missing_slots[name] = sorted(set(range(epochs * slots_per_epoch)) - seen)
        if missing_slots[name] and not allow_incomplete:
            raise ValueError(f"{name}: missing slots in epochs 0 through {epochs - 1}.")
        counts[name], rosters[name] = model_counts, roster
    if any(rosters[name] != rosters["baseline"] for name in MODELS):
        raise ValueError("Validator indexes differ across runs. Check that the runs use the same validator set.")
    return epochs, sorted(rosters["baseline"]), counts, unknown_slots, missing_slots


def write_outputs(output_dir, epochs, validators, counts, per_chart, unknown_slots=None, missing_slots=None):
    try:
        import matplotlib
        matplotlib.use("Agg")
        import matplotlib.pyplot as plt
        from matplotlib.ticker import MaxNLocator
    except ImportError as exc:
        raise ValueError("Install matplotlib first: python -m pip install matplotlib") from exc

    output_dir.mkdir(parents=True, exist_ok=True)
    with (output_dir / "validator_comparison.csv").open("w", encoding="utf-8", newline="") as stream:
        writer = csv.writer(stream)
        writer.writerow(["validator_index", *MODELS])
        for validator in validators:
            writer.writerow([validator, *(counts[name][validator] for name in MODELS)])
    ymax = max(counts[name][validator] for name in MODELS for validator in validators)
    chunk_size = per_chart or len(validators)
    for start in range(0, len(validators), chunk_size):
        selected = validators[start:start + chunk_size]
        fig, ax = plt.subplots(figsize=(max(10, len(selected) * 0.45), 6))
        width = 0.2
        for index, (name, color) in enumerate(zip(MODELS, COLORS)):
            positions = [x + (index - 1.5) * width for x in range(len(selected))]
            ax.bar(positions, [counts[name][v] for v in selected], width=width,
                   label=name.upper(), color=color)
        ax.set_xticks(range(len(selected)), [str(v) for v in selected], rotation=90)
        ax.set_xlabel("Validator index")
        ax.set_ylabel("Proposer assignment count")
        incomplete = any((unknown_slots or {}).values()) or any((missing_slots or {}).values())
        title = f"Observed proposer assignments | Epochs 0–{epochs - 1}"
        if incomplete:
            title += "\nIncomplete data: missing or unknown proposer slots excluded"
        ax.set_title(title)
        ax.set_ylim(0, max(1, ymax * 1.12))
        ax.yaxis.set_major_locator(MaxNLocator(integer=True))
        ax.grid(axis="y", alpha=0.25)
        ax.set_axisbelow(True)
        ax.legend(ncol=4)
        fig.tight_layout()
        stem = f"validators_{selected[0]}_{selected[-1]}"
        for extension in ("png", "pdf"):
            fig.savefig(output_dir / f"{stem}.{extension}", dpi=200)
        plt.close(fig)
    metadata = {"models": list(MODELS), "epochs": epochs,
                "validator_count": len(validators), "metric": "proposer assignments",
                "excluded_unknown_proposer_slots": unknown_slots or {},
                "missing_slots": missing_slots or {},
                "counted_assignments": {name: sum(counts[name].values()) for name in MODELS}}
    (output_dir / "comparison_status.json").write_text(
        json.dumps(metadata, indent=2) + "\n", encoding="utf-8")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input-dir", type=Path, default=Path("output"))
    parser.add_argument("--output-dir", type=Path, default=Path("output/comparison"))
    parser.add_argument("--epochs", type=int, help="Default: largest complete window shared by all four exports")
    parser.add_argument("--allow-incomplete", action="store_true",
                        help="Plot an explicit epoch window despite missing slots; report missing data")
    parser.add_argument("--run", action="append", default=[], metavar="MODEL=DIRECTORY",
                        help="Select an export when multiple runs exist; may be repeated")
    parser.add_argument("--validators-per-chart", type=int, default=30, help="0: all validators in one chart")
    args = parser.parse_args()
    if args.validators_per_chart < 0:
        parser.error("--validators-per-chart must be nonnegative")
    selected_runs = {}
    for selection in args.run:
        name, separator, directory = selection.partition("=")
        if not separator or name not in MODELS or not directory or name in selected_runs:
            parser.error("--run requires a unique MODEL=DIRECTORY; models: " + ", ".join(MODELS))
        selected_runs[name] = Path(directory)
    try:
        epochs, validators, counts, unknown_slots, missing_slots = load_comparison(
            args.input_dir, args.epochs, selected_runs, args.allow_incomplete)
        write_outputs(args.output_dir, epochs, validators, counts, args.validators_per_chart,
                      unknown_slots, missing_slots)
    except (ValueError, KeyError, OSError) as exc:
        parser.exit(1, f"Error: {exc}\n")
    print(f"Compared {len(validators)} validators over epochs 0–{epochs - 1}.")
    for name, slots in unknown_slots.items():
        if slots:
            print(f"Warning: {name}: excluded {len(slots)} slots with unknown proposer; see comparison_status.json.")
    for name, slots in missing_slots.items():
        if slots:
            print(f"Warning: {name}: {len(slots)} missing slots; see comparison_status.json.")
    print(f"CSV, PNG, PDF: {args.output_dir.resolve()}")


if __name__ == "__main__":
    main()
