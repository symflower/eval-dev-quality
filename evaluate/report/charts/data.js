// loadCSVFile loads a CSV file and returns the list of raw records.
async function loadCSVFile(filepath) {
    try {
        const records = d3.csv(filepath);

        return records;
    } catch (error) {
        console.error("error loading CSV file: ", error);
    }
}

// modelsWithCostsAndScores returns an array of objects with model names, costs and scores.
function modelsWithCostsAndScores(evaluationRecords, metaInformationRecords) {
    // Group models and sum scores.
    const modelsWithSummedScores = {};
    evaluationRecords.forEach((record) => {
        const modelId = record["model-id"];
        const score = +record["score"];

        if (!modelsWithSummedScores[modelId]) {
            modelsWithSummedScores[modelId] = score;
        } else {
            modelsWithSummedScores[modelId] += score;
        }
    });

    // Add costs and human-readable name.
    const models = {};
    for (const modelId in modelsWithSummedScores) {
        models[modelId] = {
            "model-name": modelId,
            "model-costs": 0.0,
            score: modelsWithSummedScores[modelId],
        };
    }
    metaInformationRecords.forEach((record) => {
        const modelId = record["model-id"];
        if (models[modelId]) {
            models[modelId]["model-name"] = record["model-name"];
            models[modelId]["model-costs"] =
                +record.completion +
                +record.image +
                +record.prompt +
                +record.request;
        }
    });

    return Object.values(models);
}

export { loadCSVFile, modelsWithCostsAndScores };
